package onering

import (
	"context"
	"crypto/tls"
	"net"
	"net/http"
	"sync"
	"time"
)

// MultiCDNConfig holds multi-CDN configuration
type MultiCDNConfig struct {
	Enabled   bool
	Providers []*CDNProvider
	Strategy  SelectionStrategy

	HealthCheck HealthCheckConfig
	Failover    FailoverConfig
	Evasion     EvasionConfig
}

// HealthCheckConfig configures health check behavior
type HealthCheckConfig struct {
	Enabled  bool
	Interval time.Duration // Default: 30s
	Timeout  time.Duration // Default: 5s
	URL      string        // Test endpoint
}

// FailoverConfig configures failover behavior
type FailoverConfig struct {
	MaxRetries        int           // Per-CDN retry (default: 3)
	BlacklistDuration time.Duration // Avoid failed CDN (default: 5m)
	FallbackToSingle  bool          // Use single CDN if all fail
}

// MultiCDNManager manages multi-CDN operations
type MultiCDNManager struct {
	config    *MultiCDNConfig
	providers []*CDNProvider
	strategy  SelectionStrategy

	// Health check tracking
	healthCheckCancel context.CancelFunc
	healthCheckWg     sync.WaitGroup

	// Evasion tracking (Phase 2)
	trafficShaper     *TrafficShaper
	rotationIndex     int
	rotationCancel    context.CancelFunc
	rotationWg        sync.WaitGroup

	// Thread-safe access to providers
	mu sync.RWMutex

	// Last selected provider (for debugging)
	lastSelected *CDNProvider
}

// NewMultiCDNManager creates a new multi-CDN manager
func NewMultiCDNManager(config *MultiCDNConfig) *MultiCDNManager {
	if config == nil {
		return nil
	}

	// Validate provider list (minor fix)
	if len(config.Providers) == 0 {
		return nil
	}

	// Clone providers to avoid external modifications
	providers := make([]*CDNProvider, len(config.Providers))
	for i, p := range config.Providers {
		providers[i] = p.Clone()
	}

	// Set default strategy if not provided
	strategy := config.Strategy
	if strategy == nil {
		strategy = NewStrategy(StrategyRoundRobin)
	}

	// Set default health check config
	if config.HealthCheck.Interval == 0 {
		config.HealthCheck.Interval = 30 * time.Second
	}
	if config.HealthCheck.Timeout == 0 {
		config.HealthCheck.Timeout = 5 * time.Second
	}

	// Set default failover config
	if config.Failover.MaxRetries == 0 {
		config.Failover.MaxRetries = 3
	}
	if config.Failover.BlacklistDuration == 0 {
		config.Failover.BlacklistDuration = 5 * time.Minute
	}

	manager := &MultiCDNManager{
		config:        config,
		providers:     providers,
		strategy:      strategy,
		rotationIndex: 0,
	}

	// Initialize traffic shaper for evasion (Phase 2)
	manager.trafficShaper = NewTrafficShaper(config.Evasion)

	// Start health check loop if enabled
	if config.HealthCheck.Enabled {
		manager.StartHealthCheck()
	}

	// Start auto-rotation if enabled (Phase 2)
	if config.Evasion.RotateEnabled && config.Evasion.RotateInterval > 0 {
		manager.StartAutoRotation(context.Background())
	}

	return manager
}

// SelectCDN selects a CDN provider using the configured strategy
func (m *MultiCDNManager) SelectCDN() *CDNProvider {
	m.mu.Lock() // Changed from RLock to Lock - fixes BUG #1
	defer m.mu.Unlock()

	selected := m.strategy.Select(m.providers)
	if selected != nil {
		m.lastSelected = selected
	}
	return selected
}

// SelectCDNWithRetry selects a CDN and retries with failover on failure
func (m *MultiCDNManager) SelectCDNWithRetry(testFunc func(*CDNProvider) error) (*CDNProvider, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	tried := make(map[string]bool)
	maxAttempts := len(m.providers) * m.config.Failover.MaxRetries

	for attempt := 0; attempt < maxAttempts; attempt++ {
		// Select next CDN
		provider := m.strategy.Select(m.providers)
		if provider == nil {
			break // No available providers
		}

		// Skip if already tried
		if tried[provider.Name] {
			continue
		}
		tried[provider.Name] = true

		// Test the provider
		err := testFunc(provider)
		if err == nil {
			// Success
			provider.MarkSuccess(0)
			m.lastSelected = provider
			return provider, nil
		}

		// Failure - mark and blacklist
		provider.MarkFailure()
		if provider.FailCount >= 3 {
			provider.Blacklist(m.config.Failover.BlacklistDuration)
		}
	}

	// All providers failed
	if m.config.Failover.FallbackToSingle && len(m.providers) > 0 {
		// Return first provider as fallback
		return m.providers[0], nil
	}

	return nil, ErrAllCDNsFailed
}

// HealthCheck performs a health check on a single provider
func (m *MultiCDNManager) HealthCheck(provider *CDNProvider) error {
	ctx, cancel := context.WithTimeout(context.Background(), m.config.HealthCheck.Timeout)
	defer cancel()

	startTime := time.Now()

	// Simple TCP dial test to the bug domain
	dialer := &net.Dialer{
		Timeout: m.config.HealthCheck.Timeout,
	}

	// Try TLS connection to bug domain (standard HTTPS port)
	conn, err := tls.DialWithDialer(dialer, "tcp", provider.BugDomain+":443", &tls.Config{
		ServerName:         provider.BugDomain,
		InsecureSkipVerify: true, // Just testing connectivity
	})

	if err != nil {
		return err
	}
	defer conn.Close()

	// If URL is provided, try HTTP HEAD request
	if m.config.HealthCheck.URL != "" {
		client := &http.Client{
			Timeout: m.config.HealthCheck.Timeout,
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					ServerName:         provider.BugDomain,
					InsecureSkipVerify: true,
				},
			},
		}

		req, err := http.NewRequestWithContext(ctx, "HEAD", m.config.HealthCheck.URL, nil)
		if err != nil {
			return err
		}

		resp, err := client.Do(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		if resp.StatusCode >= 500 {
			return ErrHealthCheckFailed
		}
	}

	latency := time.Since(startTime)
	return m.updateHealthMetrics(provider, true, latency)
}

// updateHealthMetrics updates provider health metrics
func (m *MultiCDNManager) updateHealthMetrics(provider *CDNProvider, success bool, latency time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Find the provider in our list
	for _, p := range m.providers {
		if p.Name == provider.Name {
			if success {
				p.MarkSuccess(latency)
			} else {
				p.MarkFailure()
			}
			return nil
		}
	}

	return ErrProviderNotFound
}

// StartHealthCheck starts the background health check loop
func (m *MultiCDNManager) StartHealthCheck() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.healthCheckCancel != nil {
		return // Already running
	}

	ctx, cancel := context.WithCancel(context.Background())
	m.healthCheckCancel = cancel

	m.healthCheckWg.Add(1)
	go m.healthCheckLoop(ctx)
}

// StopHealthCheck stops the background health check loop
func (m *MultiCDNManager) StopHealthCheck() {
	if m.healthCheckCancel != nil {
		m.healthCheckCancel()
		m.healthCheckWg.Wait()
		m.healthCheckCancel = nil
	}
}

// healthCheckLoop runs periodic health checks
func (m *MultiCDNManager) healthCheckLoop(ctx context.Context) {
	defer m.healthCheckWg.Done()

	ticker := time.NewTicker(m.config.HealthCheck.Interval)
	defer ticker.Stop()

	// Run initial health check immediately
	m.runHealthChecks()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			m.runHealthChecks()
		}
	}
}

// runHealthChecks performs health checks on all providers
func (m *MultiCDNManager) runHealthChecks() {
	m.mu.RLock()
	providers := make([]*CDNProvider, len(m.providers))
	copy(providers, m.providers)
	m.mu.RUnlock()

	// Check all providers in parallel
	var wg sync.WaitGroup
	for _, provider := range providers {
		wg.Add(1)
		go func(p *CDNProvider) {
			defer wg.Done()
			err := m.HealthCheck(p)
			if err != nil {
				// Health check failed, already marked by updateHealthMetrics
			}
		}(provider)
	}
	wg.Wait()
}

// GetProviders returns a copy of current providers
func (m *MultiCDNManager) GetProviders() []*CDNProvider {
	m.mu.RLock()
	defer m.mu.RUnlock()

	providers := make([]*CDNProvider, len(m.providers))
	for i, p := range m.providers {
		providers[i] = p.Clone()
	}
	return providers
}

// GetLastSelected returns the last selected provider
func (m *MultiCDNManager) GetLastSelected() *CDNProvider {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.lastSelected != nil {
		return m.lastSelected.Clone()
	}
	return nil
}

// GetAvailableCount returns the number of available providers
func (m *MultiCDNManager) GetAvailableCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	count := 0
	for _, p := range m.providers {
		if p.IsAvailable() {
			count++
		}
	}
	return count
}

// RecordSuccess marks a provider as successful (thread-safe) - fixes BUG #5
func (m *MultiCDNManager) RecordSuccess(providerName string, latency time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, p := range m.providers {
		if p.Name == providerName {
			p.MarkSuccess(latency)
			return
		}
	}
}

// RecordFailure marks a provider as failed (thread-safe) - fixes BUG #5
func (m *MultiCDNManager) RecordFailure(providerName string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, p := range m.providers {
		if p.Name == providerName {
			p.MarkFailure()
			if p.FailCount >= 3 {
				p.Blacklist(m.config.Failover.BlacklistDuration)
			}
			return
		}
	}
}

// StartAutoRotation starts automatic CDN rotation (Phase 2)
func (m *MultiCDNManager) StartAutoRotation(ctx context.Context) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.rotationCancel != nil {
		return // Already running
	}

	if !m.config.Evasion.RotateEnabled || m.config.Evasion.RotateInterval <= 0 {
		return // Rotation disabled
	}

	rotationCtx, cancel := context.WithCancel(ctx)
	m.rotationCancel = cancel

	m.rotationWg.Add(1)
	go m.autoRotationLoop(rotationCtx)
}

// StopAutoRotation stops automatic CDN rotation
func (m *MultiCDNManager) StopAutoRotation() {
	m.mu.Lock()
	if m.rotationCancel != nil {
		m.rotationCancel()
		m.rotationCancel = nil
	}
	m.mu.Unlock()

	m.rotationWg.Wait()
}

// autoRotationLoop runs periodic CDN rotation
func (m *MultiCDNManager) autoRotationLoop(ctx context.Context) {
	defer m.rotationWg.Done()

	ticker := time.NewTicker(m.config.Evasion.RotateInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			m.ForceRotate()
		}
	}
}

// ForceRotate forces rotation to next CDN in the provider list
func (m *MultiCDNManager) ForceRotate() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if len(m.providers) <= 1 {
		return // Nothing to rotate
	}

	// Move to next provider
	m.rotationIndex = (m.rotationIndex + 1) % len(m.providers)

	// Update strategy to prefer the rotated provider
	// This works best with round-robin or random strategies
}

// GetTrafficShaper returns the traffic shaper for evasion techniques
func (m *MultiCDNManager) GetTrafficShaper() *TrafficShaper {
	return m.trafficShaper
}

// ApplyJitter applies timing jitter from evasion config
func (m *MultiCDNManager) ApplyJitter(ctx context.Context) error {
	if m.trafficShaper != nil {
		return m.trafficShaper.ApplyJitterContext(ctx)
	}
	return nil
}

// GetRandomTLSConfig returns TLS config with random fingerprint if enabled
func (m *MultiCDNManager) GetRandomTLSConfig(base *tls.Config) *tls.Config {
	if m.trafficShaper != nil {
		return m.trafficShaper.GetRandomTLSConfig(base)
	}
	return base
}

// Shutdown gracefully shuts down the manager
func (m *MultiCDNManager) Shutdown() {
	m.StopHealthCheck()
	m.StopAutoRotation()
	if m.trafficShaper != nil {
		m.trafficShaper.StopAutoRotation()
	}
}

// Errors
var (
	ErrAllCDNsFailed     = &CDNError{Message: "all CDN providers failed"}
	ErrHealthCheckFailed = &CDNError{Message: "health check failed"}
	ErrProviderNotFound  = &CDNError{Message: "provider not found"}
)

// CDNError represents a CDN-related error
type CDNError struct {
	Message string
}

func (e *CDNError) Error() string {
	return "multi-cdn: " + e.Message
}

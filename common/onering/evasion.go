package onering

import (
	"context"
	"crypto/rand"
	"crypto/tls"
	"math/big"
	"sync"
	"time"
)

// EvasionConfig configures DPI evasion techniques (Phase 2)
type EvasionConfig struct {
	// Timing jitter - add random delays to avoid pattern detection
	JitterEnabled bool
	JitterMin     time.Duration // Default: 50ms
	JitterMax     time.Duration // Default: 200ms

	// Packet padding - add random payload size to avoid signature detection
	PaddingEnabled bool
	MaxPaddingSize int // Default: 512 bytes

	// CDN auto-rotation - rotate CDN provider periodically
	RotateEnabled  bool
	RotateInterval time.Duration // Default: 5 minutes

	// Random TLS fingerprint - vary ALPN and cipher suites per connection
	RandomizeTLS bool
}

// DefaultEvasionConfig returns conservative default settings
func DefaultEvasionConfig() EvasionConfig {
	return EvasionConfig{
		JitterEnabled:  false, // Disabled by default
		JitterMin:      50 * time.Millisecond,
		JitterMax:      200 * time.Millisecond,
		PaddingEnabled: false, // Disabled by default
		MaxPaddingSize: 512,
		RotateEnabled:  false, // Disabled by default
		RotateInterval: 5 * time.Minute,
		RandomizeTLS:   false, // Disabled by default
	}
}

// TrafficShaper applies DPI evasion techniques to traffic
type TrafficShaper struct {
	config         EvasionConfig
	mu             sync.RWMutex
	rotationTicker *time.Ticker
	rotationCancel context.CancelFunc
	rotationWg     sync.WaitGroup
	onRotate       func() // Callback when rotation occurs
}

// NewTrafficShaper creates a new traffic shaper with evasion config
func NewTrafficShaper(config EvasionConfig) *TrafficShaper {
	return &TrafficShaper{
		config: config,
	}
}

// ApplyJitter returns a random delay duration within configured range
// Used to add timing randomization to avoid pattern detection
func (ts *TrafficShaper) ApplyJitter(ctx context.Context) time.Duration {
	ts.mu.RLock()
	defer ts.mu.RUnlock()

	if !ts.config.JitterEnabled {
		return 0
	}

	min := ts.config.JitterMin
	max := ts.config.JitterMax

	if max <= min {
		return min
	}

	// Generate cryptographically secure random delay
	deltaMs := max - min
	if deltaMs <= 0 {
		return min
	}

	// Convert to milliseconds for random generation
	deltaMillis := int64(deltaMs / time.Millisecond)
	if deltaMillis <= 0 {
		return min
	}

	randomMs, err := rand.Int(rand.Reader, big.NewInt(deltaMillis))
	if err != nil {
		// Fallback to min on error
		return min
	}

	return min + time.Duration(randomMs.Int64())*time.Millisecond
}

// ApplyPadding adds random padding to data to avoid packet size fingerprinting
// Returns a new byte slice with original data + random padding
func (ts *TrafficShaper) ApplyPadding(data []byte) []byte {
	ts.mu.RLock()
	defer ts.mu.RUnlock()

	if !ts.config.PaddingEnabled || ts.config.MaxPaddingSize <= 0 {
		return data
	}

	// Generate random padding size (0 to MaxPaddingSize)
	maxSize := int64(ts.config.MaxPaddingSize)
	randomSize, err := rand.Int(rand.Reader, big.NewInt(maxSize+1))
	if err != nil {
		// Fallback to no padding on error
		return data
	}

	paddingSize := int(randomSize.Int64())
	if paddingSize == 0 {
		return data
	}

	// Create padding with random bytes
	padding := make([]byte, paddingSize)
	_, err = rand.Read(padding)
	if err != nil {
		// Fallback to no padding on error
		return data
	}

	// Append padding to original data
	result := make([]byte, len(data)+paddingSize)
	copy(result, data)
	copy(result[len(data):], padding)

	return result
}

// ShouldRotateCDN checks if it's time to rotate CDN provider
// This is called internally by the rotation ticker
func (ts *TrafficShaper) ShouldRotateCDN() bool {
	ts.mu.RLock()
	defer ts.mu.RUnlock()

	return ts.config.RotateEnabled && ts.config.RotateInterval > 0
}

// GetRandomTLSConfig returns a TLS config with randomized fingerprint
// Randomizes ALPN protocols and cipher suite order to avoid detection
func (ts *TrafficShaper) GetRandomTLSConfig(base *tls.Config) *tls.Config {
	ts.mu.RLock()
	defer ts.mu.RUnlock()

	if !ts.config.RandomizeTLS {
		return base
	}

	// Clone base config to avoid modifying original
	config := base.Clone()

	// Randomize ALPN order
	alpnVariants := [][]string{
		{"h2", "http/1.1"},
		{"http/1.1", "h2"},
		{"h2"},
		{"http/1.1"},
	}

	variantIndex, err := rand.Int(rand.Reader, big.NewInt(int64(len(alpnVariants))))
	if err == nil {
		config.NextProtos = alpnVariants[variantIndex.Int64()]
	}

	// Randomize cipher suite order (keep only secure ones)
	// Using recommended cipher suites from TLS 1.2/1.3
	secureCiphers := []uint16{
		tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
		tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
		tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
		tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
		tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305_SHA256,
		tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305_SHA256,
	}

	// Fisher-Yates shuffle with crypto/rand
	shuffledCiphers := make([]uint16, len(secureCiphers))
	copy(shuffledCiphers, secureCiphers)

	for i := len(shuffledCiphers) - 1; i > 0; i-- {
		jBig, err := rand.Int(rand.Reader, big.NewInt(int64(i+1)))
		if err != nil {
			break
		}
		j := int(jBig.Int64())
		shuffledCiphers[i], shuffledCiphers[j] = shuffledCiphers[j], shuffledCiphers[i]
	}

	config.CipherSuites = shuffledCiphers

	return config
}

// StartAutoRotation starts a background goroutine that triggers CDN rotation
func (ts *TrafficShaper) StartAutoRotation(ctx context.Context, onRotate func()) {
	ts.mu.Lock()
	defer ts.mu.Unlock()

	// Don't start if already running or disabled
	if ts.rotationCancel != nil || !ts.config.RotateEnabled || ts.config.RotateInterval <= 0 {
		return
	}

	// Create cancellable context
	rotationCtx, cancel := context.WithCancel(ctx)
	ts.rotationCancel = cancel
	ts.onRotate = onRotate

	// Create ticker
	ts.rotationTicker = time.NewTicker(ts.config.RotateInterval)

	// Start rotation goroutine
	ts.rotationWg.Add(1)
	go ts.rotationLoop(rotationCtx)
}

// StopAutoRotation stops the background rotation goroutine
func (ts *TrafficShaper) StopAutoRotation() {
	ts.mu.Lock()
	if ts.rotationCancel != nil {
		ts.rotationCancel()
		ts.rotationCancel = nil
	}
	if ts.rotationTicker != nil {
		ts.rotationTicker.Stop()
		ts.rotationTicker = nil
	}
	ts.mu.Unlock()

	// Wait for goroutine to finish
	ts.rotationWg.Wait()
}

// rotationLoop is the background goroutine that triggers periodic rotation
func (ts *TrafficShaper) rotationLoop(ctx context.Context) {
	defer ts.rotationWg.Done()

	// Get ticker reference at start to avoid nil pointer dereference
	ts.mu.RLock()
	ticker := ts.rotationTicker
	ts.mu.RUnlock()

	if ticker == nil {
		return // No ticker, exit immediately
	}

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// Trigger rotation callback
			ts.mu.RLock()
			onRotate := ts.onRotate
			ts.mu.RUnlock()

			if onRotate != nil {
				onRotate()
			}
		}
	}
}

// UpdateConfig updates the evasion config at runtime (thread-safe)
func (ts *TrafficShaper) UpdateConfig(config EvasionConfig) {
	ts.mu.Lock()
	defer ts.mu.Unlock()

	oldRotateEnabled := ts.config.RotateEnabled
	oldRotateInterval := ts.config.RotateInterval

	ts.config = config

	// Restart rotation if config changed
	if oldRotateEnabled != config.RotateEnabled || oldRotateInterval != config.RotateInterval {
		if ts.rotationTicker != nil {
			ts.rotationTicker.Stop()
			ts.rotationTicker = nil
		}

		if config.RotateEnabled && config.RotateInterval > 0 && ts.rotationCancel != nil {
			ts.rotationTicker = time.NewTicker(config.RotateInterval)
		}
	}
}

// GetConfig returns a copy of current evasion config (thread-safe)
func (ts *TrafficShaper) GetConfig() EvasionConfig {
	ts.mu.RLock()
	defer ts.mu.RUnlock()
	return ts.config
}

// ApplyJitterContext applies jitter delay to the current context
// Returns error if context is cancelled during jitter delay
func (ts *TrafficShaper) ApplyJitterContext(ctx context.Context) error {
	jitter := ts.ApplyJitter(ctx)
	if jitter <= 0 {
		return nil
	}

	timer := time.NewTimer(jitter)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

// TrafficStats tracks evasion technique usage statistics
type TrafficStats struct {
	mu                 sync.RWMutex
	TotalJitters       int64
	TotalJitterTime    time.Duration
	TotalPaddings      int64
	TotalPaddingBytes  int64
	TotalRotations     int64
	TotalTLSRandomized int64
}

// NewTrafficStats creates a new traffic statistics tracker
func NewTrafficStats() *TrafficStats {
	return &TrafficStats{}
}

// RecordJitter records jitter application
func (ts *TrafficStats) RecordJitter(duration time.Duration) {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	ts.TotalJitters++
	ts.TotalJitterTime += duration
}

// RecordPadding records padding application
func (ts *TrafficStats) RecordPadding(bytes int) {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	ts.TotalPaddings++
	ts.TotalPaddingBytes += int64(bytes)
}

// RecordRotation records CDN rotation
func (ts *TrafficStats) RecordRotation() {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	ts.TotalRotations++
}

// RecordTLSRandomization records TLS fingerprint randomization
func (ts *TrafficStats) RecordTLSRandomization() {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	ts.TotalTLSRandomized++
}

// GetStats returns a copy of current statistics (thread-safe)
func (ts *TrafficStats) GetStats() (jitters int64, jitterTime time.Duration, paddings int64, paddingBytes int64, rotations int64, tlsRandomized int64) {
	ts.mu.RLock()
	defer ts.mu.RUnlock()
	return ts.TotalJitters, ts.TotalJitterTime, ts.TotalPaddings, ts.TotalPaddingBytes, ts.TotalRotations, ts.TotalTLSRandomized
}

// Reset resets all statistics to zero
func (ts *TrafficStats) Reset() {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	ts.TotalJitters = 0
	ts.TotalJitterTime = 0
	ts.TotalPaddings = 0
	ts.TotalPaddingBytes = 0
	ts.TotalRotations = 0
	ts.TotalTLSRandomized = 0
}

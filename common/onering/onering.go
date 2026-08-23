package onering

import (
	"errors"
	"fmt"
	"strings"
	"sync"
)

// Prefix for onering format
const (
	Prefix          = "onering:"
	MultiCDNPrefix  = "onering-multi:"
)

// Config holds parsed onering configuration
type Config struct {
	Enabled    bool
	RealDomain string
	BugDomain  string
	
	// Multi-CDN support
	MultiCDNEnabled bool
	MultiCDNManager *MultiCDNManager
	
	// Provider selection cache - fixes BUG #3
	selectedProvider *CDNProvider
	selectionMutex   sync.RWMutex
}

// Parse parses onering format string
// Formats:
//   - Single CDN (old): "onering:real.domain.com:bug.domain.com"
//   - Multi-CDN (old): "onering-multi:real.domain.com"
//   - Multi-CDN (new SNI): "onering=bug-zoom.us,ruangguru=ruangguru.com,zenius=zenius.net,host.com"
// Returns Config or error if invalid format
func Parse(input string) (*Config, error) {
	// Empty input = disabled
	if input == "" {
		return &Config{Enabled: false}, nil
	}

	// Check for new comma-separated Multi-CDN format
	if strings.Contains(input, ",") {
		return parseMultiCDNFromSNI(input)
	}

	// Check for old multi-CDN format
	if strings.HasPrefix(input, MultiCDNPrefix) {
		return parseMultiCDN(input)
	}

	// Check for single-CDN format
	if strings.HasPrefix(input, Prefix) {
		return parseSingleCDN(input)
	}

	// Not onering format = disabled (backward compatible)
	return &Config{
		Enabled:    false,
		RealDomain: input,
		BugDomain:  "",
	}, nil
}

// parseSingleCDN parses single-CDN format: "onering:real:bug"
func parseSingleCDN(input string) (*Config, error) {
	// Remove prefix
	trimmed := strings.TrimPrefix(input, Prefix)

	// Split by ":"
	parts := strings.Split(trimmed, ":")
	if len(parts) != 2 {
		return nil, errors.New("invalid onering format: expected 'onering:real:bug'")
	}

	// Check for invalid chars in raw parts BEFORE trimming (to catch embedded newlines/control chars)
	if containsInvalidChars(parts[0]) || containsInvalidChars(parts[1]) {
		return nil, errors.New("domain contains invalid characters")
	}

	// Trim spaces from each part (only spaces, not control chars)
	real := strings.TrimSpace(parts[0])
	bug := strings.TrimSpace(parts[1])

	// Validation after trim
	if real == "" {
		return nil, errors.New("real domain cannot be empty")
	}
	if bug == "" {
		return nil, errors.New("bug domain cannot be empty")
	}

	return &Config{
		Enabled:         true,
		RealDomain:      real,
		BugDomain:       bug,
		MultiCDNEnabled: false,
	}, nil
}

// parseMultiCDN parses multi-CDN format: "onering-multi:real"
func parseMultiCDN(input string) (*Config, error) {
	// Remove prefix
	trimmed := strings.TrimPrefix(input, MultiCDNPrefix)

	// Check for invalid chars
	if containsInvalidChars(trimmed) {
		return nil, errors.New("domain contains invalid characters")
	}

	// Trim spaces
	real := strings.TrimSpace(trimmed)

	// Validation
	if real == "" {
		return nil, errors.New("real domain cannot be empty")
	}

	return &Config{
		Enabled:         true,
		RealDomain:      real,
		BugDomain:       "", // Will be selected by MultiCDNManager
		MultiCDNEnabled: true,
		MultiCDNManager: nil, // Will be set later by TLS config
	}, nil
}

// parseMultiCDNFromSNI parses comma-separated Multi-CDN format from SNI field
// Format: "onering=bug-zoom.us,ruangguru=ruangguru.com,zenius=zenius.net,host.com"
// - Multiple CDNs separated by comma ","
// - Each CDN has format: "label=domain" or just "domain"
// - Label is optional (can be "onering=", "ruangguru=", etc.)
// - Last value after comma is the actual server host (e.g., "host.com")
// - Space after comma is trimmed
func parseMultiCDNFromSNI(input string) (*Config, error) {
	// Split by comma
	parts := strings.Split(input, ",")
	if len(parts) < 2 {
		// Not enough parts for Multi-CDN format
		return nil, errors.New("invalid Multi-CDN format: expected at least 2 comma-separated values")
	}

	// Last part is the server host (real domain)
	realDomain := strings.TrimSpace(parts[len(parts)-1])
	
	// Validate real domain
	if realDomain == "" {
		return nil, errors.New("real domain (last value) cannot be empty")
	}
	if containsInvalidChars(realDomain) {
		return nil, errors.New("real domain contains invalid characters")
	}

	// Parse CDN entries (all except last)
	var cdnProviders []*CDNProvider
	for i := 0; i < len(parts)-1; i++ {
		part := strings.TrimSpace(parts[i])
		
		// Skip empty entries
		if part == "" {
			continue
		}

		var label, cdnDomain string
		
		// Check if has label: "onering=bug-zoom.us"
		if strings.Contains(part, "=") {
			kv := strings.SplitN(part, "=", 2)
			label = strings.TrimSpace(kv[0])
			cdnDomain = strings.TrimSpace(kv[1])
		} else {
			// No label, just domain
			label = fmt.Sprintf("cdn%d", i+1) // Auto-generate label
			cdnDomain = part
		}

		// Validate CDN domain
		if cdnDomain == "" {
			return nil, fmt.Errorf("CDN domain at position %d cannot be empty", i+1)
		}
		if containsInvalidChars(cdnDomain) {
			return nil, fmt.Errorf("CDN domain at position %d contains invalid characters", i+1)
		}

		// Create CDN provider
		priority := 100 - (i * 10) // Descending priority: 100, 90, 80, ...
		if priority < 10 {
			priority = 10
		}

		cdnProviders = append(cdnProviders, &CDNProvider{
			Name:       label,
			BugDomain:  cdnDomain,
			Priority:   priority,
			ISPs:       []string{}, // Available for all ISPs
			Healthy:    true,
			FailCount:  0,
			AvgLatency: 0,
		})
	}

	// Validate we have at least one CDN
	if len(cdnProviders) == 0 {
		return nil, errors.New("no valid CDN providers found")
	}

	// Create Multi-CDN manager with parsed providers
	multiCDNConfig := &MultiCDNConfig{
		Enabled:   true,
		Providers: cdnProviders,
		Strategy:  NewStrategy(StrategyRoundRobin), // Default to round-robin
		HealthCheck: HealthCheckConfig{
			Enabled:  false, // Disabled by default for SNI-based config
			Interval: 30 * 1000000000, // 30 seconds
			Timeout:  5 * 1000000000,  // 5 seconds
		},
		Failover: FailoverConfig{
			MaxRetries:        3,
			BlacklistDuration: 5 * 60 * 1000000000, // 5 minutes
			FallbackToSingle:  true,
		},
	}

	manager := NewMultiCDNManager(multiCDNConfig)

	return &Config{
		Enabled:         true,
		RealDomain:      realDomain,
		BugDomain:       "", // Will be selected dynamically
		MultiCDNEnabled: true,
		MultiCDNManager: manager,
	}, nil
}

// containsInvalidChars checks for characters that shouldn't be in domain names
func containsInvalidChars(domain string) bool {
	// Reject control characters (but allow space 32 since it will be trimmed)
	for _, r := range domain {
		if (r < 32 || r == 127) {
			return true
		}
	}
	// Reject common injection patterns (but not space)
	return strings.ContainsAny(domain, "\r\n\t\"'<>")
}

// selectProviderOnce selects and caches provider for this connection - fixes BUG #3
func (c *Config) selectProviderOnce() *CDNProvider {
	c.selectionMutex.Lock()
	defer c.selectionMutex.Unlock()
	
	if c.selectedProvider == nil && c.MultiCDNManager != nil {
		c.selectedProvider = c.MultiCDNManager.SelectCDN()
	}
	return c.selectedProvider
}

// GetDialAddress returns the address to dial (bug domain if enabled)
func (c *Config) GetDialAddress() string {
	if c.Enabled {
		// Multi-CDN: select once and cache - fixes BUG #3
		if c.MultiCDNEnabled && c.MultiCDNManager != nil {
			provider := c.selectProviderOnce()
			if provider != nil {
				return provider.BugDomain
			}
		}
		// Single-CDN or fallback
		if c.BugDomain != "" {
			return c.BugDomain
		}
	}
	return c.RealDomain
}

// GetTLSSNI returns SNI for TLS handshake
func (c *Config) GetTLSSNI() string {
	if c.Enabled {
		// Multi-CDN: use cached provider - fixes BUG #3
		if c.MultiCDNEnabled && c.MultiCDNManager != nil {
			provider := c.selectProviderOnce()
			if provider != nil {
				return provider.BugDomain
			}
		}
		// Single-CDN or fallback
		if c.BugDomain != "" {
			return c.BugDomain
		}
	}
	return c.RealDomain
}

// GetHTTPHost returns Host header for HTTP/WebSocket
func (c *Config) GetHTTPHost() string {
	return c.RealDomain
}

// String returns human-readable format
func (c *Config) String() string {
	if !c.Enabled {
		return "onering:disabled"
	}
	if c.MultiCDNEnabled {
		availableCount := 0
		if c.MultiCDNManager != nil {
			availableCount = c.MultiCDNManager.GetAvailableCount()
		}
		// Fixed string conversion bug - fixes BUG #4
		return fmt.Sprintf("onering:multi-cdn(real=%s,providers=%d)", c.RealDomain, availableCount)
	}
	return "onering:enabled(real=" + c.RealDomain + ",bug=" + c.BugDomain + ")"
}

// SetMultiCDNManager sets the multi-CDN manager for this config
func (c *Config) SetMultiCDNManager(manager *MultiCDNManager) {
	c.MultiCDNManager = manager
}

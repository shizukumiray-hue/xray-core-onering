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
//   - Single CDN: "onering:real.domain.com:bug.domain.com"
//   - Multi-CDN: "onering-multi:real.domain.com"
// Returns Config or error if invalid format
func Parse(input string) (*Config, error) {
	// Empty input = disabled
	if input == "" {
		return &Config{Enabled: false}, nil
	}

	// Check for multi-CDN format first
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

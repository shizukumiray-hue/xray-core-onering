package onering

import (
	"errors"
	"strings"
)

// Prefix for onering format
const Prefix = "onering:"

// Config holds parsed onering configuration
type Config struct {
	Enabled    bool
	RealDomain string
	BugDomain  string
}

// Parse parses onering format string
// Format: "onering:real.domain.com:bug.domain.com"
// Returns Config or error if invalid format
func Parse(input string) (*Config, error) {
	// Empty input = disabled
	if input == "" {
		return &Config{Enabled: false}, nil
	}

	// Not onering format = disabled (backward compatible)
	if !strings.HasPrefix(input, Prefix) {
		return &Config{
			Enabled:    false,
			RealDomain: input,
			BugDomain:  "",
		}, nil
	}

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
		Enabled:    true,
		RealDomain: real,
		BugDomain:  bug,
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

// GetDialAddress returns the address to dial (bug domain if enabled)
func (c *Config) GetDialAddress() string {
	if c.Enabled {
		return c.BugDomain
	}
	return c.RealDomain
}

// GetTLSSNI returns SNI for TLS handshake
func (c *Config) GetTLSSNI() string {
	if c.Enabled {
		return c.BugDomain
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
	return "onering:enabled(real=" + c.RealDomain + ",bug=" + c.BugDomain + ")"
}

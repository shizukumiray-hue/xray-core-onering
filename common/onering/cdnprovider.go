package onering

import (
	"time"
)

// CDNProvider represents a CDN provider configuration and runtime status
type CDNProvider struct {
	Name       string   // "cloudflare", "cloudfront", "fastly", "akamai", "gcore"
	BugDomain  string   // SNI bug domain for this CDN
	Priority   int      // 1-100 (higher = preferred)
	ISPs       []string // Target ISPs (e.g., ["telkomsel", "indosat"]) or empty for all

	// Runtime status (managed by health check system)
	Healthy      bool
	LastCheck    time.Time
	FailCount    int
	SuccessCount int
	AvgLatency   time.Duration

	// Blacklist tracking
	Blacklisted       bool
	BlacklistUntil    time.Time
	BlacklistDuration time.Duration
}

// HealthMetrics holds health check metrics for a CDN provider
type HealthMetrics struct {
	SuccessRate   float64       // 0.0-1.0
	AvgLatency    time.Duration
	HealthScore   float64       // 0.0-1.0 (weighted score)
	LastCheckTime time.Time
}

// GetHealthMetrics calculates current health metrics for this provider
func (p *CDNProvider) GetHealthMetrics() HealthMetrics {
	totalChecks := p.SuccessCount + p.FailCount
	successRate := 0.0
	if totalChecks > 0 {
		successRate = float64(p.SuccessCount) / float64(totalChecks)
	}

	// Normalize latency (assume 0-1000ms range)
	normalizedLatency := 0.0
	if p.AvgLatency > 0 {
		normalizedLatency = float64(p.AvgLatency.Milliseconds()) / 1000.0
		if normalizedLatency > 1.0 {
			normalizedLatency = 1.0
		}
	}

	// Health score: 70% success rate + 30% latency
	healthScore := (successRate * 0.7) + ((1.0 - normalizedLatency) * 0.3)

	return HealthMetrics{
		SuccessRate:   successRate,
		AvgLatency:    p.AvgLatency,
		HealthScore:   healthScore,
		LastCheckTime: p.LastCheck,
	}
}

// IsAvailable checks if provider is available for selection
func (p *CDNProvider) IsAvailable() bool {
	// Check if blacklisted
	if p.Blacklisted && time.Now().Before(p.BlacklistUntil) {
		return false
	}

	// Clear blacklist if expired
	if p.Blacklisted && time.Now().After(p.BlacklistUntil) {
		p.Blacklisted = false
	}

	return p.Healthy
}

// MarkSuccess updates provider status after successful connection
func (p *CDNProvider) MarkSuccess(latency time.Duration) {
	p.Healthy = true
	p.FailCount = 0
	p.SuccessCount++
	p.LastCheck = time.Now()

	// Update average latency (rolling average)
	if p.AvgLatency == 0 {
		p.AvgLatency = latency
	} else {
		// Simple exponential moving average
		p.AvgLatency = (p.AvgLatency*9 + latency) / 10
	}

	// Clear blacklist on success
	if p.Blacklisted {
		p.Blacklisted = false
	}
}

// MarkFailure updates provider status after failed connection
func (p *CDNProvider) MarkFailure() {
	p.FailCount++
	p.LastCheck = time.Now()

	// Mark unhealthy after 3 consecutive failures
	if p.FailCount >= 3 {
		p.Healthy = false
	}
}

// Blacklist marks this provider as blacklisted for the configured duration
func (p *CDNProvider) Blacklist(duration time.Duration) {
	p.Blacklisted = true
	p.BlacklistDuration = duration
	p.BlacklistUntil = time.Now().Add(duration)
	p.Healthy = false
}

// Clone creates a deep copy of the provider
func (p *CDNProvider) Clone() *CDNProvider {
	isps := make([]string, len(p.ISPs))
	copy(isps, p.ISPs)

	return &CDNProvider{
		Name:              p.Name,
		BugDomain:         p.BugDomain,
		Priority:          p.Priority,
		ISPs:              isps,
		Healthy:           p.Healthy,
		LastCheck:         p.LastCheck,
		FailCount:         p.FailCount,
		SuccessCount:      p.SuccessCount,
		AvgLatency:        p.AvgLatency,
		Blacklisted:       p.Blacklisted,
		BlacklistUntil:    p.BlacklistUntil,
		BlacklistDuration: p.BlacklistDuration,
	}
}

// DefaultCDNProviders returns a list of default CDN providers
func DefaultCDNProviders() []*CDNProvider {
	return []*CDNProvider{
		{
			Name:       "cloudflare",
			BugDomain:  "zoom.us",
			Priority:   100,
			ISPs:       []string{"telkomsel", "indosat"},
			Healthy:    true,
			FailCount:  0,
			AvgLatency: 0,
		},
		{
			Name:       "cloudfront",
			BugDomain:  "aws.amazon.com",
			Priority:   90,
			ISPs:       []string{"xl", "3"},
			Healthy:    true,
			FailCount:  0,
			AvgLatency: 0,
		},
		{
			Name:       "fastly",
			BugDomain:  "wa.me",
			Priority:   80,
			ISPs:       []string{"indosat", "xl"},
			Healthy:    true,
			FailCount:  0,
			AvgLatency: 0,
		},
		{
			Name:       "akamai",
			BugDomain:  "facebook.com",
			Priority:   70,
			ISPs:       []string{"telkomsel"},
			Healthy:    true,
			FailCount:  0,
			AvgLatency: 0,
		},
		{
			Name:       "gcore",
			BugDomain:  "discord.com",
			Priority:   60,
			ISPs:       []string{}, // Available for all ISPs
			Healthy:    true,
			FailCount:  0,
			AvgLatency: 0,
		},
	}
}

// MatchesISP checks if this provider supports the given ISP
func (p *CDNProvider) MatchesISP(isp string) bool {
	// Empty ISPs list means available for all ISPs
	if len(p.ISPs) == 0 {
		return true
	}

	// Check if ISP is in the list
	for _, supportedISP := range p.ISPs {
		if supportedISP == isp {
			return true
		}
	}

	return false
}

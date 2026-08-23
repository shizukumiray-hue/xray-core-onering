package onering

import (
	"math/rand"
	"sync"
	"sync/atomic"
	"time"
)

// SelectionStrategy defines the interface for CDN selection strategies
type SelectionStrategy interface {
	// Select chooses a CDN provider from the available list
	Select(providers []*CDNProvider) *CDNProvider
	// Name returns the strategy name
	Name() string
}

// StrategyType represents different selection strategies
type StrategyType int

const (
	StrategyRoundRobin   StrategyType = 0 // Rotate evenly
	StrategyFailover     StrategyType = 1 // Primary + backup
	StrategyLatencyBased StrategyType = 2 // Choose fastest
	StrategyHealthBased  StrategyType = 3 // Weight by health score
	StrategyRandom       StrategyType = 4 // Random (for evasion)
)

// NewStrategy creates a strategy instance based on type
func NewStrategy(strategyType StrategyType) SelectionStrategy {
	switch strategyType {
	case StrategyRoundRobin:
		return &RoundRobinStrategy{}
	case StrategyFailover:
		return &FailoverStrategy{}
	case StrategyLatencyBased:
		return &LatencyBasedStrategy{}
	case StrategyHealthBased:
		return &HealthBasedStrategy{}
	case StrategyRandom:
		return &RandomStrategy{}
	default:
		return &RoundRobinStrategy{}
	}
}

// ParseStrategyType converts string to StrategyType
func ParseStrategyType(s string) StrategyType {
	switch s {
	case "round-robin", "roundrobin":
		return StrategyRoundRobin
	case "failover":
		return StrategyFailover
	case "latency-based", "latency":
		return StrategyLatencyBased
	case "health-based", "health":
		return StrategyHealthBased
	case "random":
		return StrategyRandom
	default:
		return StrategyRoundRobin
	}
}

// RoundRobinStrategy cycles through providers evenly
type RoundRobinStrategy struct {
	counter uint64
}

func (s *RoundRobinStrategy) Name() string {
	return "round-robin"
}

func (s *RoundRobinStrategy) Select(providers []*CDNProvider) *CDNProvider {
	if len(providers) == 0 {
		return nil
	}

	// Filter available providers
	available := filterAvailable(providers)
	if len(available) == 0 {
		return nil
	}

	// Atomic increment and modulo
	index := atomic.AddUint64(&s.counter, 1) - 1
	return available[index%uint64(len(available))]
}

// FailoverStrategy uses primary, switches on failure
type FailoverStrategy struct{}

func (s *FailoverStrategy) Name() string {
	return "failover"
}

func (s *FailoverStrategy) Select(providers []*CDNProvider) *CDNProvider {
	if len(providers) == 0 {
		return nil
	}

	// Sort by priority (highest first)
	available := filterAvailable(providers)
	if len(available) == 0 {
		return nil
	}

	// Find highest priority available provider
	best := available[0]
	for _, p := range available[1:] {
		if p.Priority > best.Priority {
			best = p
		}
	}

	return best
}

// LatencyBasedStrategy selects provider with lowest latency
type LatencyBasedStrategy struct{}

func (s *LatencyBasedStrategy) Name() string {
	return "latency-based"
}

func (s *LatencyBasedStrategy) Select(providers []*CDNProvider) *CDNProvider {
	if len(providers) == 0 {
		return nil
	}

	available := filterAvailable(providers)
	if len(available) == 0 {
		return nil
	}

	// Find provider with lowest latency
	best := available[0]
	bestLatency := best.AvgLatency
	if bestLatency == 0 {
		bestLatency = time.Hour // Treat 0 as unknown (worst)
	}

	for _, p := range available[1:] {
		latency := p.AvgLatency
		if latency == 0 {
			latency = time.Hour
		}
		if latency < bestLatency {
			best = p
			bestLatency = latency
		}
	}

	return best
}

// HealthBasedStrategy weights by health score
type HealthBasedStrategy struct{}

func (s *HealthBasedStrategy) Name() string {
	return "health-based"
}

func (s *HealthBasedStrategy) Select(providers []*CDNProvider) *CDNProvider {
	if len(providers) == 0 {
		return nil
	}

	available := filterAvailable(providers)
	if len(available) == 0 {
		return nil
	}

	// Find provider with highest health score
	best := available[0]
	bestScore := best.GetHealthMetrics().HealthScore

	for _, p := range available[1:] {
		score := p.GetHealthMetrics().HealthScore
		if score > bestScore {
			best = p
			bestScore = score
		}
	}

	return best
}

// RandomStrategy selects random provider (for DPI evasion)
type RandomStrategy struct {
	mu  sync.Mutex // Added mutex - fixes BUG #2
	rng *rand.Rand
}

func (s *RandomStrategy) Name() string {
	return "random"
}

func (s *RandomStrategy) Select(providers []*CDNProvider) *CDNProvider {
	if len(providers) == 0 {
		return nil
	}

	available := filterAvailable(providers)
	if len(available) == 0 {
		return nil
	}

	// Thread-safe random selection - fixes BUG #2
	s.mu.Lock()
	if s.rng == nil {
		s.rng = rand.New(rand.NewSource(time.Now().UnixNano()))
	}
	index := s.rng.Intn(len(available))
	s.mu.Unlock()

	return available[index]
}

// filterAvailable returns only available (healthy and not blacklisted) providers
func filterAvailable(providers []*CDNProvider) []*CDNProvider {
	available := make([]*CDNProvider, 0, len(providers))
	for _, p := range providers {
		if p.IsAvailable() {
			available = append(available, p)
		}
	}
	return available
}

package onering

import (
	"context"
	"crypto/tls"
	"sync"
	"testing"
	"time"
)

func TestDefaultEvasionConfig(t *testing.T) {
	cfg := DefaultEvasionConfig()

	// All features should be disabled by default
	if cfg.JitterEnabled {
		t.Error("JitterEnabled should be false by default")
	}
	if cfg.PaddingEnabled {
		t.Error("PaddingEnabled should be false by default")
	}
	if cfg.RotateEnabled {
		t.Error("RotateEnabled should be false by default")
	}
	if cfg.RandomizeTLS {
		t.Error("RandomizeTLS should be false by default")
	}

	// Check default values
	if cfg.JitterMin != 50*time.Millisecond {
		t.Errorf("JitterMin should be 50ms, got %v", cfg.JitterMin)
	}
	if cfg.JitterMax != 200*time.Millisecond {
		t.Errorf("JitterMax should be 200ms, got %v", cfg.JitterMax)
	}
	if cfg.MaxPaddingSize != 512 {
		t.Errorf("MaxPaddingSize should be 512, got %d", cfg.MaxPaddingSize)
	}
	if cfg.RotateInterval != 5*time.Minute {
		t.Errorf("RotateInterval should be 5m, got %v", cfg.RotateInterval)
	}
}

func TestApplyJitter_Disabled(t *testing.T) {
	config := EvasionConfig{
		JitterEnabled: false,
		JitterMin:     50 * time.Millisecond,
		JitterMax:     200 * time.Millisecond,
	}

	shaper := NewTrafficShaper(config)
	ctx := context.Background()

	jitter := shaper.ApplyJitter(ctx)
	if jitter != 0 {
		t.Errorf("Jitter should be 0 when disabled, got %v", jitter)
	}
}

func TestApplyJitter_Enabled(t *testing.T) {
	config := EvasionConfig{
		JitterEnabled: true,
		JitterMin:     50 * time.Millisecond,
		JitterMax:     200 * time.Millisecond,
	}

	shaper := NewTrafficShaper(config)
	ctx := context.Background()

	// Test multiple times to ensure randomness
	for i := 0; i < 10; i++ {
		jitter := shaper.ApplyJitter(ctx)

		if jitter < config.JitterMin {
			t.Errorf("Jitter %v is less than min %v", jitter, config.JitterMin)
		}
		if jitter > config.JitterMax {
			t.Errorf("Jitter %v is greater than max %v", jitter, config.JitterMax)
		}
	}
}

func TestApplyJitter_Randomness(t *testing.T) {
	config := EvasionConfig{
		JitterEnabled: true,
		JitterMin:     10 * time.Millisecond,
		JitterMax:     100 * time.Millisecond,
	}

	shaper := NewTrafficShaper(config)
	ctx := context.Background()

	// Collect 20 samples
	samples := make(map[time.Duration]bool)
	for i := 0; i < 20; i++ {
		jitter := shaper.ApplyJitter(ctx)
		samples[jitter] = true
	}

	// Should have at least some variation (not all the same)
	if len(samples) < 3 {
		t.Error("Jitter values are not random enough")
	}
}

func TestApplyPadding_Disabled(t *testing.T) {
	config := EvasionConfig{
		PaddingEnabled: false,
		MaxPaddingSize: 512,
	}

	shaper := NewTrafficShaper(config)
	data := []byte("test data")

	result := shaper.ApplyPadding(data)

	if len(result) != len(data) {
		t.Errorf("Padding should not be applied when disabled, got %d bytes, expected %d", len(result), len(data))
	}
}

func TestApplyPadding_Enabled(t *testing.T) {
	config := EvasionConfig{
		PaddingEnabled: true,
		MaxPaddingSize: 256,
	}

	shaper := NewTrafficShaper(config)
	data := []byte("test data")

	// Test multiple times
	foundPadded := false

	for i := 0; i < 20; i++ {
		result := shaper.ApplyPadding(data)

		// Should be >= original size
		if len(result) < len(data) {
			t.Errorf("Result is smaller than original data: %d < %d", len(result), len(data))
		}

		// Should be <= original + max padding
		if len(result) > len(data)+config.MaxPaddingSize {
			t.Errorf("Result is larger than max: %d > %d", len(result), len(data)+config.MaxPaddingSize)
		}

		// Original data should be preserved at the beginning
		for j := 0; j < len(data); j++ {
			if result[j] != data[j] {
				t.Errorf("Original data corrupted at position %d", j)
			}
		}

		if len(result) > len(data) {
			foundPadded = true
		}
	}

	// Should have randomness (some with padding, some without)
	if !foundPadded {
		t.Error("No padding was ever applied (unlikely with 20 tries)")
	}
}

func TestApplyPadding_PreservesData(t *testing.T) {
	config := EvasionConfig{
		PaddingEnabled: true,
		MaxPaddingSize: 128,
	}

	shaper := NewTrafficShaper(config)
	originalData := []byte("important data that must not be corrupted")

	for i := 0; i < 10; i++ {
		result := shaper.ApplyPadding(originalData)

		// Original data should be at the start
		if len(result) < len(originalData) {
			t.Fatal("Result is smaller than original")
		}

		for j := 0; j < len(originalData); j++ {
			if result[j] != originalData[j] {
				t.Errorf("Data corrupted at position %d: got %d, expected %d", j, result[j], originalData[j])
			}
		}
	}
}

func TestGetRandomTLSConfig_Disabled(t *testing.T) {
	config := EvasionConfig{
		RandomizeTLS: false,
	}

	shaper := NewTrafficShaper(config)
	baseConfig := &tls.Config{
		NextProtos: []string{"h2", "http/1.1"},
	}

	result := shaper.GetRandomTLSConfig(baseConfig)

	// Should return the same config (not randomized)
	if result != baseConfig {
		t.Error("Should return the same config when randomization is disabled")
	}
}

func TestGetRandomTLSConfig_Enabled(t *testing.T) {
	config := EvasionConfig{
		RandomizeTLS: true,
	}

	shaper := NewTrafficShaper(config)
	baseConfig := &tls.Config{
		NextProtos: []string{"h2", "http/1.1"},
	}

	// Test multiple times to check randomness
	alpnVariations := make(map[string]bool)
	cipherVariations := make(map[string]bool)

	for i := 0; i < 20; i++ {
		result := shaper.GetRandomTLSConfig(baseConfig)

		// Should be a different instance (cloned)
		if result == baseConfig {
			t.Error("Should return a cloned config, not the original")
		}

		// Record ALPN variation
		if len(result.NextProtos) > 0 {
			alpnKey := ""
			for _, proto := range result.NextProtos {
				alpnKey += proto + ","
			}
			alpnVariations[alpnKey] = true
		}

		// Record cipher variation
		if len(result.CipherSuites) > 0 {
			cipherKey := ""
			for _, cipher := range result.CipherSuites {
				cipherKey += string(rune(cipher)) + ","
			}
			cipherVariations[cipherKey] = true
		}

		// Verify cipher suites are secure (from our predefined list)
		secureCiphers := map[uint16]bool{
			tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256:         true,
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384:         true,
			tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256:       true,
			tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384:       true,
			tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305_SHA256:   true,
			tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305_SHA256: true,
		}

		for _, cipher := range result.CipherSuites {
			if !secureCiphers[cipher] {
				t.Errorf("Insecure cipher suite used: %x", cipher)
			}
		}
	}

	// Should have at least 2 ALPN variations
	if len(alpnVariations) < 2 {
		t.Error("Not enough ALPN variations (expected randomness)")
	}

	// Should have multiple cipher order variations
	if len(cipherVariations) < 2 {
		t.Error("Not enough cipher suite variations (expected randomness)")
	}
}

func TestAutoRotation_Disabled(t *testing.T) {
	config := EvasionConfig{
		RotateEnabled:  false,
		RotateInterval: 100 * time.Millisecond,
	}

	shaper := NewTrafficShaper(config)
	ctx := context.Background()

	callCount := 0
	onRotate := func() {
		callCount++
	}

	shaper.StartAutoRotation(ctx, onRotate)
	defer shaper.StopAutoRotation()

	// Wait longer than rotation interval
	time.Sleep(250 * time.Millisecond)

	// Should not have called rotation callback
	if callCount > 0 {
		t.Errorf("Rotation callback called %d times when disabled", callCount)
	}
}

func TestAutoRotation_Enabled(t *testing.T) {
	config := EvasionConfig{
		RotateEnabled:  true,
		RotateInterval: 100 * time.Millisecond,
	}

	shaper := NewTrafficShaper(config)
	ctx := context.Background()

	callCount := 0
	onRotate := func() {
		callCount++
	}

	shaper.StartAutoRotation(ctx, onRotate)
	defer shaper.StopAutoRotation()

	// Wait for ~2.5 intervals
	time.Sleep(250 * time.Millisecond)

	// Should have called rotation callback at least once
	if callCount < 1 {
		t.Errorf("Rotation callback should be called at least once, got %d", callCount)
	}

	// Should not have called too many times (allow some timing variance)
	if callCount > 4 {
		t.Errorf("Rotation callback called too many times: %d", callCount)
	}
}

func TestAutoRotation_Cancel(t *testing.T) {
	config := EvasionConfig{
		RotateEnabled:  true,
		RotateInterval: 50 * time.Millisecond,
	}

	shaper := NewTrafficShaper(config)
	ctx, cancel := context.WithCancel(context.Background())

	var mu sync.Mutex
	callCount := 0
	onRotate := func() {
		mu.Lock()
		callCount++
		mu.Unlock()
	}

	shaper.StartAutoRotation(ctx, onRotate)

	// Wait for first rotation
	time.Sleep(100 * time.Millisecond)
	mu.Lock()
	firstCount := callCount
	mu.Unlock()

	// Cancel context and stop
	cancel()
	shaper.StopAutoRotation()

	// Record count immediately after stop
	mu.Lock()
	countAfterStop := callCount
	mu.Unlock()

	// Wait more time
	time.Sleep(150 * time.Millisecond)
	mu.Lock()
	finalCount := callCount
	mu.Unlock()

	// Should have rotated at least once before cancel
	if firstCount < 1 {
		t.Error("Should have rotated before cancel")
	}

	// Should not have rotated after stop completed
	if finalCount != countAfterStop {
		t.Errorf("Should not rotate after stop: afterStop=%d, final=%d", countAfterStop, finalCount)
	}
}

func TestApplyJitterContext_Cancel(t *testing.T) {
	config := EvasionConfig{
		JitterEnabled: true,
		JitterMin:     500 * time.Millisecond,
		JitterMax:     1000 * time.Millisecond,
	}

	shaper := NewTrafficShaper(config)
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	start := time.Now()
	err := shaper.ApplyJitterContext(ctx)
	elapsed := time.Since(start)

	// Should return error due to context cancellation
	if err == nil {
		t.Error("Expected error due to context timeout")
	}

	// Should have returned quickly (within timeout + margin)
	if elapsed > 200*time.Millisecond {
		t.Errorf("Took too long to cancel: %v", elapsed)
	}
}

func TestApplyJitterContext_Success(t *testing.T) {
	config := EvasionConfig{
		JitterEnabled: true,
		JitterMin:     10 * time.Millisecond,
		JitterMax:     50 * time.Millisecond,
	}

	shaper := NewTrafficShaper(config)
	ctx := context.Background()

	start := time.Now()
	err := shaper.ApplyJitterContext(ctx)
	elapsed := time.Since(start)

	// Should not return error
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// Should have waited at least min jitter
	if elapsed < config.JitterMin {
		t.Errorf("Waited too little: %v < %v", elapsed, config.JitterMin)
	}

	// Should not have waited more than max jitter + margin
	if elapsed > config.JitterMax+50*time.Millisecond {
		t.Errorf("Waited too long: %v > %v", elapsed, config.JitterMax)
	}
}

func TestTrafficStats(t *testing.T) {
	stats := NewTrafficStats()

	// Record some stats
	stats.RecordJitter(100 * time.Millisecond)
	stats.RecordJitter(200 * time.Millisecond)
	stats.RecordPadding(128)
	stats.RecordPadding(256)
	stats.RecordRotation()
	stats.RecordRotation()
	stats.RecordRotation()
	stats.RecordTLSRandomization()

	// Check stats
	jitters, jitterTime, paddings, paddingBytes, rotations, tlsRand := stats.GetStats()

	if jitters != 2 {
		t.Errorf("Expected 2 jitters, got %d", jitters)
	}
	if jitterTime != 300*time.Millisecond {
		t.Errorf("Expected 300ms total jitter time, got %v", jitterTime)
	}
	if paddings != 2 {
		t.Errorf("Expected 2 paddings, got %d", paddings)
	}
	if paddingBytes != 384 {
		t.Errorf("Expected 384 padding bytes, got %d", paddingBytes)
	}
	if rotations != 3 {
		t.Errorf("Expected 3 rotations, got %d", rotations)
	}
	if tlsRand != 1 {
		t.Errorf("Expected 1 TLS randomization, got %d", tlsRand)
	}

	// Reset stats
	stats.Reset()

	jitters, jitterTime, paddings, paddingBytes, rotations, tlsRand = stats.GetStats()
	if jitters != 0 || paddings != 0 || rotations != 0 || tlsRand != 0 {
		t.Error("Stats should be zero after reset")
	}
}

func TestUpdateConfig(t *testing.T) {
	initialConfig := EvasionConfig{
		JitterEnabled:  false,
		PaddingEnabled: false,
	}

	shaper := NewTrafficShaper(initialConfig)

	// Verify initial state
	if shaper.GetConfig().JitterEnabled {
		t.Error("JitterEnabled should be false initially")
	}

	// Update config
	newConfig := EvasionConfig{
		JitterEnabled:  true,
		JitterMin:      10 * time.Millisecond,
		JitterMax:      50 * time.Millisecond,
		PaddingEnabled: true,
		MaxPaddingSize: 256,
	}

	shaper.UpdateConfig(newConfig)

	// Verify updated config
	updatedConfig := shaper.GetConfig()
	if !updatedConfig.JitterEnabled {
		t.Error("JitterEnabled should be true after update")
	}
	if !updatedConfig.PaddingEnabled {
		t.Error("PaddingEnabled should be true after update")
	}
	if updatedConfig.MaxPaddingSize != 256 {
		t.Errorf("MaxPaddingSize should be 256, got %d", updatedConfig.MaxPaddingSize)
	}
}

func BenchmarkApplyJitter(b *testing.B) {
	config := EvasionConfig{
		JitterEnabled: true,
		JitterMin:     10 * time.Millisecond,
		JitterMax:     50 * time.Millisecond,
	}

	shaper := NewTrafficShaper(config)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = shaper.ApplyJitter(ctx)
	}
}

func BenchmarkApplyPadding(b *testing.B) {
	config := EvasionConfig{
		PaddingEnabled: true,
		MaxPaddingSize: 512,
	}

	shaper := NewTrafficShaper(config)
	data := make([]byte, 1024)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = shaper.ApplyPadding(data)
	}
}

func BenchmarkGetRandomTLSConfig(b *testing.B) {
	config := EvasionConfig{
		RandomizeTLS: true,
	}

	shaper := NewTrafficShaper(config)
	baseConfig := &tls.Config{
		NextProtos: []string{"h2", "http/1.1"},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = shaper.GetRandomTLSConfig(baseConfig)
	}
}

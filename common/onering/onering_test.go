package onering

import "testing"

func TestParse(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		wantErr    bool
		enabled    bool
		realDomain string
		bugDomain  string
	}{
		{
			name:       "valid onering format",
			input:      "onering:real.example.com:bug.example.com",
			wantErr:    false,
			enabled:    true,
			realDomain: "real.example.com",
			bugDomain:  "bug.example.com",
		},
		{
			name:    "empty input",
			input:   "",
			wantErr: false,
			enabled: false,
		},
		{
			name:       "normal domain (backward compatible)",
			input:      "example.com",
			wantErr:    false,
			enabled:    false,
			realDomain: "example.com",
		},
		{
			name:    "invalid format - missing bug domain",
			input:   "onering:real.example.com",
			wantErr: true,
		},
		{
			name:    "invalid format - too many colons",
			input:   "onering:real:bug:extra",
			wantErr: true,
		},
		{
			name:    "invalid characters - newline",
			input:   "onering:real.com\n:bug.com",
			wantErr: true,
		},
		{
			name:       "with spaces (trimmed)",
			input:      "onering: real.com : bug.com ",
			wantErr:    false,
			enabled:    true,
			realDomain: "real.com",
			bugDomain:  "bug.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, err := Parse(tt.input)

			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if cfg.Enabled != tt.enabled {
				t.Errorf("Enabled = %v, want %v", cfg.Enabled, tt.enabled)
			}

			if tt.enabled {
				if cfg.RealDomain != tt.realDomain {
					t.Errorf("RealDomain = %v, want %v", cfg.RealDomain, tt.realDomain)
				}
				if cfg.BugDomain != tt.bugDomain {
					t.Errorf("BugDomain = %v, want %v", cfg.BugDomain, tt.bugDomain)
				}
			}
		})
	}
}

func TestGetMethods(t *testing.T) {
	// Test enabled config
	cfg := &Config{
		Enabled:    true,
		RealDomain: "real.com",
		BugDomain:  "bug.com",
	}

	if cfg.GetDialAddress() != "bug.com" {
		t.Errorf("GetDialAddress() should return bug domain when enabled")
	}
	if cfg.GetTLSSNI() != "bug.com" {
		t.Errorf("GetTLSSNI() should return bug domain when enabled")
	}
	if cfg.GetHTTPHost() != "real.com" {
		t.Errorf("GetHTTPHost() should return real domain when enabled")
	}

	// Test disabled config
	cfg2 := &Config{
		Enabled:    false,
		RealDomain: "example.com",
	}

	if cfg2.GetDialAddress() != "example.com" {
		t.Errorf("GetDialAddress() should return real domain when disabled")
	}
	if cfg2.GetHTTPHost() != "example.com" {
		t.Errorf("GetHTTPHost() should return real domain when disabled")
	}
}

func TestParseMultiCDNFromSNI(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		wantErr       bool
		enabled       bool
		realDomain    string
		multiCDN      bool
		expectedCDNs  int
		cdnDomains    []string
		cdnLabels     []string
	}{
		{
			name:         "Multi-CDN with labels",
			input:        "onering=zoom.us,ruangguru=ruangguru.com,zenius=zenius.net,server.com",
			wantErr:      false,
			enabled:      true,
			realDomain:   "server.com",
			multiCDN:     true,
			expectedCDNs: 3,
			cdnDomains:   []string{"zoom.us", "ruangguru.com", "zenius.net"},
			cdnLabels:    []string{"onering", "ruangguru", "zenius"},
		},
		{
			name:         "Multi-CDN without labels",
			input:        "zoom.us,ruangguru.com,zenius.net,server.com",
			wantErr:      false,
			enabled:      true,
			realDomain:   "server.com",
			multiCDN:     true,
			expectedCDNs: 3,
			cdnDomains:   []string{"zoom.us", "ruangguru.com", "zenius.net"},
			cdnLabels:    []string{"cdn1", "cdn2", "cdn3"},
		},
		{
			name:         "Multi-CDN mixed (with and without labels)",
			input:        "onering=zoom.us,ruangguru.com,zenius=zenius.net,server.com",
			wantErr:      false,
			enabled:      true,
			realDomain:   "server.com",
			multiCDN:     true,
			expectedCDNs: 3,
			cdnDomains:   []string{"zoom.us", "ruangguru.com", "zenius.net"},
			cdnLabels:    []string{"onering", "cdn2", "zenius"},
		},
		{
			name:         "Multi-CDN with spaces (trimmed)",
			input:        "onering=zoom.us , ruangguru=ruangguru.com , server.com ",
			wantErr:      false,
			enabled:      true,
			realDomain:   "server.com",
			multiCDN:     true,
			expectedCDNs: 2,
			cdnDomains:   []string{"zoom.us", "ruangguru.com"},
			cdnLabels:    []string{"onering", "ruangguru"},
		},
		{
			name:         "Two CDNs minimum",
			input:        "zoom.us,server.com",
			wantErr:      false,
			enabled:      true,
			realDomain:   "server.com",
			multiCDN:     true,
			expectedCDNs: 1,
			cdnDomains:   []string{"zoom.us"},
			cdnLabels:    []string{"cdn1"},
		},
		{
			name:    "Invalid - single value (no comma)",
			input:   "server.com",
			wantErr: false, // Falls back to plain domain
			enabled: false,
		},
		{
			name:    "Invalid - empty real domain",
			input:   "onering=zoom.us,",
			wantErr: true,
		},
		{
			name:    "Invalid - empty CDN domain",
			input:   "onering=,server.com",
			wantErr: true,
		},
		{
			name:    "Invalid - contains newline in CDN domain",
			input:   "onering=zoom.us\nmalicious,server.com",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, err := Parse(tt.input)

			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if cfg.Enabled != tt.enabled {
				t.Errorf("Enabled = %v, want %v", cfg.Enabled, tt.enabled)
			}

			if !tt.enabled {
				return // Skip further checks for disabled configs
			}

			if cfg.RealDomain != tt.realDomain {
				t.Errorf("RealDomain = %v, want %v", cfg.RealDomain, tt.realDomain)
			}

			if cfg.MultiCDNEnabled != tt.multiCDN {
				t.Errorf("MultiCDNEnabled = %v, want %v", cfg.MultiCDNEnabled, tt.multiCDN)
			}

			if tt.multiCDN {
				if cfg.MultiCDNManager == nil {
					t.Errorf("MultiCDNManager should not be nil for Multi-CDN config")
					return
				}

				providers := cfg.MultiCDNManager.GetProviders()
				if len(providers) != tt.expectedCDNs {
					t.Errorf("Expected %d CDN providers, got %d", tt.expectedCDNs, len(providers))
				}

				// Verify CDN domains
				for i, provider := range providers {
					if i >= len(tt.cdnDomains) {
						break
					}
					if provider.BugDomain != tt.cdnDomains[i] {
						t.Errorf("CDN %d: BugDomain = %v, want %v", i, provider.BugDomain, tt.cdnDomains[i])
					}
					if provider.Name != tt.cdnLabels[i] {
						t.Errorf("CDN %d: Name = %v, want %v", i, provider.Name, tt.cdnLabels[i])
					}
				}

				// Test CDN selection
				selectedCDN := cfg.GetDialAddress()
				if selectedCDN == "" {
					t.Errorf("GetDialAddress() should return a selected CDN domain")
				}

				// Verify it's one of the configured CDN domains
				found := false
				for _, domain := range tt.cdnDomains {
					if selectedCDN == domain {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("GetDialAddress() returned %v, which is not in configured CDN domains %v", selectedCDN, tt.cdnDomains)
				}
			}
		})
	}
}

func TestBackwardCompatibility(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		wantErr    bool
		enabled    bool
		realDomain string
		bugDomain  string
		multiCDN   bool
	}{
		{
			name:       "Old single-CDN format",
			input:      "onering:server.com:zoom.us",
			wantErr:    false,
			enabled:    true,
			realDomain: "server.com",
			bugDomain:  "zoom.us",
			multiCDN:   false,
		},
		{
			name:       "Old multi-CDN format",
			input:      "onering-multi:server.com",
			wantErr:    false,
			enabled:    true,
			realDomain: "server.com",
			bugDomain:  "",
			multiCDN:   true,
		},
		{
			name:       "New comma-separated multi-CDN format",
			input:      "onering=zoom.us,ruangguru=ruangguru.com,server.com",
			wantErr:    false,
			enabled:    true,
			realDomain: "server.com",
			multiCDN:   true,
		},
		{
			name:       "Plain domain (no Onering)",
			input:      "example.com",
			wantErr:    false,
			enabled:    false,
			realDomain: "example.com",
			multiCDN:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, err := Parse(tt.input)

			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if cfg.Enabled != tt.enabled {
				t.Errorf("Enabled = %v, want %v", cfg.Enabled, tt.enabled)
			}

			if cfg.RealDomain != tt.realDomain {
				t.Errorf("RealDomain = %v, want %v", cfg.RealDomain, tt.realDomain)
			}

			if cfg.MultiCDNEnabled != tt.multiCDN {
				t.Errorf("MultiCDNEnabled = %v, want %v", cfg.MultiCDNEnabled, tt.multiCDN)
			}

			if tt.enabled && !tt.multiCDN && tt.bugDomain != "" {
				if cfg.BugDomain != tt.bugDomain {
					t.Errorf("BugDomain = %v, want %v", cfg.BugDomain, tt.bugDomain)
				}
			}
		})
	}
}

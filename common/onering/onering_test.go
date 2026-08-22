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

package conf_test

import (
	"encoding/json"
	"strings"
	"testing"

	. "github.com/xtls/xray-core/infra/conf"
)

func TestMultiCDN_MaxProvidersLimit(t *testing.T) {
	// Create config with 100 providers (exceeds limit of 50)
	var providersJSON []string
	for i := 0; i < 100; i++ {
		providersJSON = append(providersJSON, `{"name":"provider`+string(rune('A'+i%26))+`","bugDomain":"example.com"}`)
	}

	configJSON := `{
		"serverName": "onering-multi:test.com",
		"multiCDN": {
			"enabled": true,
			"providers": [` + strings.Join(providersJSON, ",") + `]
		}
	}`

	config := new(TLSConfig)
	if err := json.Unmarshal([]byte(configJSON), config); err != nil {
		t.Fatal(err)
	}

	_, err := config.Build()
	if err == nil {
		t.Error("Expected error for exceeding max providers, got none")
	}
	if !strings.Contains(err.Error(), "too many providers") {
		t.Errorf("Expected 'too many providers' error, got: %v", err)
	}
}

func TestMultiCDN_NegativeDuration(t *testing.T) {
	configJSON := `{
		"serverName": "onering-multi:test.com",
		"multiCDN": {
			"enabled": true,
			"providers": [{"name":"test","bugDomain":"example.com"}],
			"healthCheck": {
				"interval": "-30s"
			}
		}
	}`

	config := new(TLSConfig)
	if err := json.Unmarshal([]byte(configJSON), config); err != nil {
		t.Fatal(err)
	}

	_, err := config.Build()
	if err == nil {
		t.Error("Expected error for negative interval, got none")
	}
	if !strings.Contains(err.Error(), "must be positive") {
		t.Errorf("Expected 'must be positive' error, got: %v", err)
	}
}

func TestMultiCDN_ExcessiveTimeout(t *testing.T) {
	configJSON := `{
		"serverName": "onering-multi:test.com",
		"multiCDN": {
			"enabled": true,
			"providers": [{"name":"test","bugDomain":"example.com"}],
			"healthCheck": {
				"timeout": "999999h"
			}
		}
	}`

	config := new(TLSConfig)
	if err := json.Unmarshal([]byte(configJSON), config); err != nil {
		t.Fatal(err)
	}

	_, err := config.Build()
	if err == nil {
		t.Error("Expected error for excessive timeout, got none")
	}
	if !strings.Contains(err.Error(), "too long") {
		t.Errorf("Expected 'too long' error, got: %v", err)
	}
}

func TestMultiCDN_InvalidHealthCheckURL(t *testing.T) {
	tests := []struct {
		name string
		url  string
	}{
		{"file scheme", "file:///etc/passwd"},
		{"loopback", "http://127.0.0.1/test"},
		{"private IP", "http://192.168.1.1/test"},
		{"link-local", "http://169.254.1.1/test"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			configJSON := `{
				"serverName": "onering-multi:test.com",
				"multiCDN": {
					"enabled": true,
					"providers": [{"name":"test","bugDomain":"example.com"}],
					"healthCheck": {
						"url": "` + tt.url + `"
					}
				}
			}`

			config := new(TLSConfig)
			if err := json.Unmarshal([]byte(configJSON), config); err != nil {
				t.Fatal(err)
			}

			_, err := config.Build()
			if err == nil {
				t.Errorf("Expected error for %s, got none", tt.name)
			}
		})
	}
}

func TestMultiCDN_ExcessiveRetries(t *testing.T) {
	configJSON := `{
		"serverName": "onering-multi:test.com",
		"multiCDN": {
			"enabled": true,
			"providers": [{"name":"test","bugDomain":"example.com"}],
			"failover": {
				"maxRetries": 999999
			}
		}
	}`

	config := new(TLSConfig)
	if err := json.Unmarshal([]byte(configJSON), config); err != nil {
		t.Fatal(err)
	}

	_, err := config.Build()
	if err == nil {
		t.Error("Expected error for excessive maxRetries, got none")
	}
	if !strings.Contains(err.Error(), "exceeds limit") {
		t.Errorf("Expected 'exceeds limit' error, got: %v", err)
	}
}

func TestMultiCDN_InvalidDomain(t *testing.T) {
	tests := []struct {
		name   string
		domain string
	}{
		{"special chars", "test@#$.com"},
		{"spaces", "test domain.com"},
		{"too long", strings.Repeat("a", 260) + ".com"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			configJSON := `{
				"serverName": "onering-multi:test.com",
				"multiCDN": {
					"enabled": true,
					"providers": [{"name":"test","bugDomain":"` + tt.domain + `"}]
				}
			}`

			config := new(TLSConfig)
			if err := json.Unmarshal([]byte(configJSON), config); err != nil {
				t.Fatal(err)
			}

			_, err := config.Build()
			if err == nil {
				t.Errorf("Expected error for %s, got none", tt.name)
			}
		})
	}
}

func TestMultiCDN_DuplicateProviderNames(t *testing.T) {
	configJSON := `{
		"serverName": "onering-multi:test.com",
		"multiCDN": {
			"enabled": true,
			"providers": [
				{"name":"cloudflare","bugDomain":"zoom.us"},
				{"name":"cloudflare","bugDomain":"teams.microsoft.com"}
			]
		}
	}`

	config := new(TLSConfig)
	if err := json.Unmarshal([]byte(configJSON), config); err != nil {
		t.Fatal(err)
	}

	_, err := config.Build()
	if err == nil {
		t.Error("Expected error for duplicate provider names, got none")
	}
	if !strings.Contains(err.Error(), "duplicate") {
		t.Errorf("Expected 'duplicate' error, got: %v", err)
	}
}

func TestMultiCDN_ValidConfig(t *testing.T) {
	configJSON := `{
		"serverName": "onering-multi:test.com",
		"multiCDN": {
			"enabled": true,
			"strategy": "health-based",
			"providers": [
				{"name":"cloudflare","bugDomain":"zoom.us","priority":100},
				{"name":"cloudfront","bugDomain":"aws.amazon.com","priority":90}
			],
			"healthCheck": {
				"enabled": true,
				"interval": "30s",
				"timeout": "5s",
				"url": "https://cloudflare.com/cdn-cgi/trace"
			},
			"failover": {
				"maxRetries": 3,
				"blacklistDuration": "5m"
			}
		}
	}`

	config := new(TLSConfig)
	if err := json.Unmarshal([]byte(configJSON), config); err != nil {
		t.Fatal(err)
	}

	_, err := config.Build()
	if err != nil {
		t.Errorf("Valid config should not error, got: %v", err)
	}
}

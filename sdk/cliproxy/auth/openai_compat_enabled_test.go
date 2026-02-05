package auth

import (
	"testing"

	internalconfig "github.com/router-for-me/CLIProxyAPI/v6/internal/config"
)

func TestResolveOpenAICompatConfig_EnabledFiltering(t *testing.T) {
	enabledTrue := true
	enabledFalse := false

	tests := []struct {
		name       string
		cfg        *internalconfig.Config
		compatName string
		wantName   string
		wantNil    bool
	}{
		{
			name: "enabled vendor is returned",
			cfg: &internalconfig.Config{
				OpenAICompatibility: []internalconfig.OpenAICompatibility{
					{Name: "vendor1", Enabled: &enabledTrue, BaseURL: "https://example.com"},
				},
			},
			compatName: "vendor1",
			wantName:   "vendor1",
		},
		{
			name: "nil enabled (default) vendor is returned",
			cfg: &internalconfig.Config{
				OpenAICompatibility: []internalconfig.OpenAICompatibility{
					{Name: "vendor1", Enabled: nil, BaseURL: "https://example.com"},
				},
			},
			compatName: "vendor1",
			wantName:   "vendor1",
		},
		{
			name: "disabled vendor is skipped",
			cfg: &internalconfig.Config{
				OpenAICompatibility: []internalconfig.OpenAICompatibility{
					{Name: "vendor1", Enabled: &enabledFalse, BaseURL: "https://example.com"},
				},
			},
			compatName: "vendor1",
			wantNil:    true,
		},
		{
			name: "mixed enabled/disabled returns enabled only",
			cfg: &internalconfig.Config{
				OpenAICompatibility: []internalconfig.OpenAICompatibility{
					{Name: "vendor1", Enabled: &enabledFalse, BaseURL: "https://example1.com"},
					{Name: "vendor2", Enabled: &enabledTrue, BaseURL: "https://example2.com"},
				},
			},
			compatName: "vendor2",
			wantName:   "vendor2",
		},
		{
			name: "all vendors disabled returns nil",
			cfg: &internalconfig.Config{
				OpenAICompatibility: []internalconfig.OpenAICompatibility{
					{Name: "vendor1", Enabled: &enabledFalse, BaseURL: "https://example1.com"},
					{Name: "vendor2", Enabled: &enabledFalse, BaseURL: "https://example2.com"},
				},
			},
			compatName: "vendor1",
			wantNil:    true,
		},
		{
			name: "disabled vendor with same name skipped, no fallback",
			cfg: &internalconfig.Config{
				OpenAICompatibility: []internalconfig.OpenAICompatibility{
					{Name: "vendor1", Enabled: &enabledFalse, BaseURL: "https://example.com"},
				},
			},
			compatName: "vendor1",
			wantNil:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := resolveOpenAICompatConfig(tt.cfg, "", tt.compatName, "")
			if tt.wantNil {
				if result != nil {
					t.Errorf("resolveOpenAICompatConfig() = %v, want nil", result.Name)
				}
			} else {
				if result == nil {
					t.Errorf("resolveOpenAICompatConfig() = nil, want %q", tt.wantName)
				} else if result.Name != tt.wantName {
					t.Errorf("resolveOpenAICompatConfig().Name = %q, want %q", result.Name, tt.wantName)
				}
			}
		})
	}
}

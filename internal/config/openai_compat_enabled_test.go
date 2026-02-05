package config

import (
	"testing"

	"gopkg.in/yaml.v3"
)

func TestOpenAICompatibility_IsEnabled(t *testing.T) {
	tests := []struct {
		name    string
		enabled *bool
		want    bool
	}{
		{
			name:    "nil defaults to true",
			enabled: nil,
			want:    true,
		},
		{
			name:    "explicitly true",
			enabled: boolPtr(true),
			want:    true,
		},
		{
			name:    "explicitly false",
			enabled: boolPtr(false),
			want:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			compat := OpenAICompatibility{
				Name:    "test",
				Enabled: tt.enabled,
			}
			if got := compat.IsEnabled(); got != tt.want {
				t.Errorf("IsEnabled() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestOpenAICompatibility_YAMLUnmarshal(t *testing.T) {
	tests := []struct {
		name       string
		yaml       string
		wantEnabled bool
	}{
		{
			name: "enabled field omitted defaults to true",
			yaml: `
name: test
base-url: https://example.com
models: []
`,
			wantEnabled: true,
		},
		{
			name: "enabled explicitly true",
			yaml: `
name: test
enabled: true
base-url: https://example.com
models: []
`,
			wantEnabled: true,
		},
		{
			name: "enabled explicitly false",
			yaml: `
name: test
enabled: false
base-url: https://example.com
models: []
`,
			wantEnabled: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var compat OpenAICompatibility
			if err := yaml.Unmarshal([]byte(tt.yaml), &compat); err != nil {
				t.Fatalf("yaml.Unmarshal() error = %v", err)
			}
			if got := compat.IsEnabled(); got != tt.wantEnabled {
				t.Errorf("IsEnabled() = %v, want %v", got, tt.wantEnabled)
			}
		})
	}
}

func TestOpenAICompatibility_YAMLMarshal(t *testing.T) {
	tests := []struct {
		name        string
		compat      OpenAICompatibility
		wantContain string
		wantOmit    string
	}{
		{
			name: "enabled nil omits field",
			compat: OpenAICompatibility{
				Name:    "test",
				BaseURL: "https://example.com",
				Enabled: nil,
			},
			wantOmit: "enabled",
		},
		{
			name: "enabled true includes field",
			compat: OpenAICompatibility{
				Name:    "test",
				BaseURL: "https://example.com",
				Enabled: boolPtr(true),
			},
			wantContain: "enabled: true",
		},
		{
			name: "enabled false includes field",
			compat: OpenAICompatibility{
				Name:    "test",
				BaseURL: "https://example.com",
				Enabled: boolPtr(false),
			},
			wantContain: "enabled: false",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := yaml.Marshal(&tt.compat)
			if err != nil {
				t.Fatalf("yaml.Marshal() error = %v", err)
			}
			yamlStr := string(data)
			if tt.wantContain != "" && !contains(yamlStr, tt.wantContain) {
				t.Errorf("yaml output should contain %q, got:\n%s", tt.wantContain, yamlStr)
			}
			if tt.wantOmit != "" && contains(yamlStr, tt.wantOmit) {
				t.Errorf("yaml output should not contain %q, got:\n%s", tt.wantOmit, yamlStr)
			}
		})
	}
}

func boolPtr(b bool) *bool {
	return &b
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

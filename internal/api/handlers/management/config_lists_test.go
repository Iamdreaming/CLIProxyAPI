package management

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/router-for-me/CLIProxyAPI/v6/internal/config"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func TestPatchOpenAICompat_EnabledField(t *testing.T) {
	t.Run("enable_vendor", func(t *testing.T) {
		// Create temp config file
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "config.yaml")

		// Write initial config (simple YAML without comments)
		initialYAML := `openai-compatibility:
  - name: test-provider
    base-url: https://api.test.com
    enabled: false
    models:
      - name: gpt-4
`
		if err := os.WriteFile(configPath, []byte(initialYAML), 0644); err != nil {
			t.Fatalf("failed to write config: %v", err)
		}

		cfg, _ := config.LoadConfig(configPath)
		h := &Handler{cfg: cfg, configFilePath: configPath}

		body := map[string]any{
			"name":    "test-provider",
			"value":   map[string]any{"enabled": true},
		}
		bodyBytes, _ := json.Marshal(body)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodPatch, "/openai-compatibility", bytes.NewReader(bodyBytes))
		c.Request.Header.Set("Content-Type", "application/json")

		h.PatchOpenAICompat(c)

		if w.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
		}

		if cfg.OpenAICompatibility[0].Enabled == nil || !*cfg.OpenAICompatibility[0].Enabled {
			t.Fatalf("expected vendor to be enabled")
		}
	})

	t.Run("disable_vendor", func(t *testing.T) {
		// Create temp config file
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "config.yaml")

		// Write initial config (simple YAML without comments)
		initialYAML := `openai-compatibility:
  - name: test-provider
    base-url: https://api.test.com
    enabled: true
    models:
      - name: gpt-4
`
		if err := os.WriteFile(configPath, []byte(initialYAML), 0644); err != nil {
			t.Fatalf("failed to write config: %v", err)
		}

		cfg, _ := config.LoadConfig(configPath)
		h := &Handler{cfg: cfg, configFilePath: configPath}

		body := map[string]any{
			"name":    "test-provider",
			"value":   map[string]any{"enabled": false},
		}
		bodyBytes, _ := json.Marshal(body)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodPatch, "/openai-compatibility", bytes.NewReader(bodyBytes))
		c.Request.Header.Set("Content-Type", "application/json")

		h.PatchOpenAICompat(c)

		if w.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
		}

		if cfg.OpenAICompatibility[0].Enabled == nil || *cfg.OpenAICompatibility[0].Enabled {
			t.Fatalf("expected vendor to be disabled")
		}
	})

	t.Run("preserve_enabled_when_not_provided", func(t *testing.T) {
		// Create temp config file
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "config.yaml")

		// Write initial config (simple YAML without comments)
		initialYAML := `openai-compatibility:
  - name: test-provider
    base-url: https://api.test.com
    enabled: false
    models:
      - name: gpt-4
`
		if err := os.WriteFile(configPath, []byte(initialYAML), 0644); err != nil {
			t.Fatalf("failed to write config: %v", err)
		}

		cfg, _ := config.LoadConfig(configPath)
		h := &Handler{cfg: cfg, configFilePath: configPath}

		// Only update prefix, not enabled
		body := map[string]any{
			"name":    "test-provider",
			"value":   map[string]any{"prefix": "updated-prefix"},
		}
		bodyBytes, _ := json.Marshal(body)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodPatch, "/openai-compatibility", bytes.NewReader(bodyBytes))
		c.Request.Header.Set("Content-Type", "application/json")

		h.PatchOpenAICompat(c)

		if w.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
		}

		// Enabled should still be false
		if cfg.OpenAICompatibility[0].Enabled == nil || *cfg.OpenAICompatibility[0].Enabled {
			t.Fatalf("expected vendor to remain disabled")
		}
		// Prefix should be updated
		if cfg.OpenAICompatibility[0].Prefix != "updated-prefix" {
			t.Fatalf("expected prefix to be updated")
		}
	})
}

func TestOpenAICompatibility_IsEnabled(t *testing.T) {
	t.Run("nil_enabled_is_enabled", func(t *testing.T) {
		compat := config.OpenAICompatibility{
			Name:    "test",
			BaseURL: "https://test.com",
			Enabled: nil,
		}
		if !compat.IsEnabled() {
			t.Fatalf("expected nil Enabled to be treated as enabled")
		}
	})

	t.Run("true_enabled_is_enabled", func(t *testing.T) {
		enabled := true
		compat := config.OpenAICompatibility{
			Name:    "test",
			BaseURL: "https://test.com",
			Enabled: &enabled,
		}
		if !compat.IsEnabled() {
			t.Fatalf("expected Enabled=true to be enabled")
		}
	})

	t.Run("false_enabled_is_disabled", func(t *testing.T) {
		enabled := false
		compat := config.OpenAICompatibility{
			Name:    "test",
			BaseURL: "https://test.com",
			Enabled: &enabled,
		}
		if compat.IsEnabled() {
			t.Fatalf("expected Enabled=false to be disabled")
		}
	})
}

func boolPtr(b bool) *bool {
	return &b
}


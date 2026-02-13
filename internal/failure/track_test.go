package failure

import (
	"sync"
	"testing"
	"time"

	"github.com/router-for-me/CLIProxyAPI/v6/internal/config"
)

func TestFailureTracker_TrackFailure(t *testing.T) {
	// Create a fresh tracker for this test
	tracker := &failureTracker{
		globalConfig:  nil,
		checkInterval: time.Second * 10,
		stopChan:      make(chan struct{}),
	}
	tracker.wg.Add(1)
	go tracker.autoReenableLoop()
	defer tracker.Close()

	tests := []struct {
		name           string
		vendor         string
		model          string
		failures       int
		expectDisabled bool
	}{
		{
			name:           "first failure",
			vendor:         "openai",
			model:          "gpt-4",
			failures:       1,
			expectDisabled: false,
		},
		{
			name:           "multiple failures below threshold",
			vendor:         "claude",
			model:          "claude-3-opus",
			failures:       4,
			expectDisabled: false,
		},
		{
			name:           "failure at threshold",
			vendor:         "gemini",
			model:          "gemini-pro",
			failures:       5,
			expectDisabled: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Enable the model first
			_ = tracker.EnableModel(tt.vendor, tt.model)

			// Track failures
			for i := 0; i < tt.failures; i++ {
				err := tracker.TrackFailure(tt.vendor, tt.model)
				if err != nil {
					t.Fatalf("TrackFailure returned error: %v", err)
				}
			}

			disabled, err := tracker.IsDisabled(tt.vendor, tt.model)
			if err != nil {
				t.Fatalf("IsDisabled returned error: %v", err)
			}
			if disabled != tt.expectDisabled {
				t.Errorf("IsDisabled() = %v, want %v", disabled, tt.expectDisabled)
			}
		})
	}
}

func TestFailureTracker_TrackSuccess(t *testing.T) {
	tracker := &failureTracker{
		globalConfig:  nil,
		checkInterval: time.Second * 10,
		stopChan:      make(chan struct{}),
	}
	tracker.wg.Add(1)
	go tracker.autoReenableLoop()
	defer tracker.Close()

	vendor := "openai"
	model := "gpt-4"

	// Track some failures
	for i := 0; i < 3; i++ {
		_ = tracker.TrackFailure(vendor, model)
	}

	count, _ := tracker.GetFailureCount(vendor, model)
	if count != 3 {
		t.Errorf("Expected failure count 3, got %d", count)
	}

	// Track success
	err := tracker.TrackSuccess(vendor, model)
	if err != nil {
		t.Fatalf("TrackSuccess returned error: %v", err)
	}

	count, _ = tracker.GetFailureCount(vendor, model)
	if count != 0 {
		t.Errorf("Expected failure count 0 after success, got %d", count)
	}
}

func TestFailureTracker_AutoReenable(t *testing.T) {
	cfg := &config.AutoDisableConfig{
		FailureThreshold:     3,
		TimeWindowSeconds:   60,
		DisableDurationSeconds: 2, // 2 seconds for testing
	}
	tracker := &failureTracker{
		globalConfig:  cfg,
		checkInterval: time.Second,
		stopChan:      make(chan struct{}),
	}
	tracker.wg.Add(1)
	go tracker.autoReenableLoop()
	defer tracker.Close()

	vendor := "claude"
	model := "claude-3-sonnet"

	// Disable the model
	for i := 0; i < 3; i++ {
		_ = tracker.TrackFailure(vendor, model)
	}

	disabled, _ := tracker.IsDisabled(vendor, model)
	if !disabled {
		t.Error("Expected model to be disabled after threshold failures")
	}

	// Wait for auto-reenable
	time.Sleep(3 * time.Second)

	disabled, _ = tracker.IsDisabled(vendor, model)
	if disabled {
		t.Error("Expected model to be auto-reenabled after duration")
	}
}

func TestFailureTracker_GetDisabledModels(t *testing.T) {
	tracker := &failureTracker{
		globalConfig:  nil,
		checkInterval: time.Second * 10,
		stopChan:      make(chan struct{}),
	}
	tracker.wg.Add(1)
	go tracker.autoReenableLoop()
	defer tracker.Close()

	// Disable some models
	_ = tracker.TrackFailure("openai", "gpt-4")
	_ = tracker.TrackFailure("openai", "gpt-4")
	_ = tracker.TrackFailure("openai", "gpt-4")
	_ = tracker.TrackFailure("openai", "gpt-4")
	_ = tracker.TrackFailure("openai", "gpt-4") // Now disabled

	_ = tracker.TrackFailure("claude", "claude-3-opus")
	_ = tracker.TrackFailure("claude", "claude-3-opus")
	_ = tracker.TrackFailure("claude", "claude-3-opus")
	_ = tracker.TrackFailure("claude", "claude-3-opus")
	_ = tracker.TrackFailure("claude", "claude-3-opus") // Now disabled

	disabled := tracker.GetDisabledModels()
	if len(disabled) != 2 {
		t.Errorf("Expected 2 disabled models, got %d", len(disabled))
	}
}

func TestFailureTracker_EnableModel(t *testing.T) {
	tracker := &failureTracker{
		globalConfig:  nil,
		checkInterval: time.Second * 10,
		stopChan:      make(chan struct{}),
	}
	tracker.wg.Add(1)
	go tracker.autoReenableLoop()
	defer tracker.Close()

	vendor := "gemini"
	model := "gemini-pro"

	// Disable the model
	for i := 0; i < 5; i++ {
		_ = tracker.TrackFailure(vendor, model)
	}

	disabled, _ := tracker.IsDisabled(vendor, model)
	if !disabled {
		t.Error("Expected model to be disabled")
	}

	// Manually enable
	err := tracker.EnableModel(vendor, model)
	if err != nil {
		t.Fatalf("EnableModel returned error: %v", err)
	}

	disabled, _ = tracker.IsDisabled(vendor, model)
	if disabled {
		t.Error("Expected model to be enabled after manual enable")
	}
}

func TestFailureTracker_Concurrent(t *testing.T) {
	tracker := &failureTracker{
		globalConfig:  nil,
		checkInterval: time.Second * 10,
		stopChan:      make(chan struct{}),
	}
	tracker.wg.Add(1)
	go tracker.autoReenableLoop()
	defer tracker.Close()

	vendor := "openai"
	model := "gpt-4"

	// Run concurrent failures
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 5; j++ {
				_ = tracker.TrackFailure(vendor, model)
			}
		}()
	}
	wg.Wait()

	// Check that model is disabled
	disabled, err := tracker.IsDisabled(vendor, model)
	if err != nil {
		t.Fatalf("IsDisabled returned error: %v", err)
	}
	if !disabled {
		t.Error("Expected model to be disabled after concurrent failures")
	}
}

func TestMockFailureTracker(t *testing.T) {
	tracker := NewMockFailureTracker()
	defer tracker.Close()

	// Test basic operations
	err := tracker.TrackFailure("vendor", "model")
	if err != nil {
		t.Fatalf("TrackFailure returned error: %v", err)
	}

	disabled, _ := tracker.IsDisabled("vendor", "model")
	if disabled {
		t.Error("Expected model not to be disabled with mock tracker")
	}

	// Set disabled state directly
	tracker.SetDisabled("vendor", "model", true)
	disabled, _ = tracker.IsDisabled("vendor", "model")
	if !disabled {
		t.Error("Expected model to be disabled")
	}

	// Enable model
	err = tracker.EnableModel("vendor", "model")
	if err != nil {
		t.Fatalf("EnableModel returned error: %v", err)
	}

	disabled, _ = tracker.IsDisabled("vendor", "model")
	if disabled {
		t.Error("Expected model to be enabled")
	}
}

func TestAutoDisableConfig_GetEffectiveConfig(t *testing.T) {
	// Test with nil config
	cfg := (*config.AutoDisableConfig)(nil)
	effective := cfg.GetEffectiveConfig()
	if effective.FailureThreshold != 5 {
		t.Errorf("Expected default FailureThreshold 5, got %d", effective.FailureThreshold)
	}

	// Test with zero values
	cfg = &config.AutoDisableConfig{}
	effective = cfg.GetEffectiveConfig()
	if effective.FailureThreshold != 5 {
		t.Errorf("Expected default FailureThreshold 5, got %d", effective.FailureThreshold)
	}
	if effective.TimeWindowSeconds != 60 {
		t.Errorf("Expected default TimeWindowSeconds 60, got %d", effective.TimeWindowSeconds)
	}
	if effective.DisableDurationSeconds != 300 {
		t.Errorf("Expected default DisableDurationSeconds 300, got %d", effective.DisableDurationSeconds)
	}

	// Test with custom values
	cfg = &config.AutoDisableConfig{
		FailureThreshold:     10,
		TimeWindowSeconds:   120,
		DisableDurationSeconds: 600,
	}
	effective = cfg.GetEffectiveConfig()
	if effective.FailureThreshold != 10 {
		t.Errorf("Expected FailureThreshold 10, got %d", effective.FailureThreshold)
	}
}

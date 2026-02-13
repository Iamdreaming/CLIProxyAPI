// Package failure provides automatic failure-based model disabling functionality.
package failure

import (
	"context"

	"github.com/router-for-me/CLIProxyAPI/v6/internal/config"
)

// GetEffectiveAutoDisableConfig returns the effective auto-disable configuration
// resolving from global, vendor, and model levels.
func GetEffectiveAutoDisableConfig(globalConfig *config.AutoDisableConfig, vendorConfig *config.AutoDisableConfig, modelConfig *config.AutoDisableConfig) config.AutoDisableConfig {
	// Model-level config takes priority
	if modelConfig != nil {
		return modelConfig.GetEffectiveConfig()
	}
	// Vendor-level config
	if vendorConfig != nil {
		return vendorConfig.GetEffectiveConfig()
	}
	// Global config
	if globalConfig != nil {
		return globalConfig.GetEffectiveConfig()
	}
	// Default config
	return config.AutoDisableConfig{
		FailureThreshold:     5,
		TimeWindowSeconds:    60,
		DisableDurationSeconds: 300,
	}
}

// Integration provides failure tracking integration helpers.
type Integration struct {
	tracker FailureTracker
}

// NewIntegration creates a new failure tracking integration.
func NewIntegration(tracker FailureTracker) *Integration {
	return &Integration{tracker: tracker}
}

// OnRequestStart is called when a request starts.
// It checks if the model is currently disabled and returns an error if so.
func (i *Integration) OnRequestStart(ctx context.Context, vendor, model string) (bool, error) {
	if i.tracker == nil {
		return false, nil
	}
	disabled, err := i.tracker.IsDisabled(vendor, model)
	if err != nil {
		return false, err
	}
	return disabled, nil
}

// OnRequestSuccess is called when a request succeeds.
// It resets the failure count for the model.
func (i *Integration) OnRequestSuccess(ctx context.Context, vendor, model string) error {
	if i.tracker == nil {
		return nil
	}
	return i.tracker.TrackSuccess(vendor, model)
}

// OnRequestFailure is called when a request fails.
// It increments the failure count for the model.
func (i *Integration) OnRequestFailure(ctx context.Context, vendor, model string) error {
	if i.tracker == nil {
		return nil
	}
	return i.tracker.TrackFailure(vendor, model)
}

// IsModelDisabled checks if a specific model is disabled.
func (i *Integration) IsModelDisabled(vendor, model string) (bool, error) {
	if i.tracker == nil {
		return false, nil
	}
	return i.tracker.IsDisabled(vendor, model)
}

// GetAllDisabledModels returns all currently disabled models.
func (i *Integration) GetAllDisabledModels() []DisabledModel {
	if i.tracker == nil {
		return nil
	}
	return i.tracker.GetDisabledModels()
}

// EnableModel manually re-enables a disabled model.
func (i *Integration) EnableModel(vendor, model string) error {
	if i.tracker == nil {
		return nil
	}
	return i.tracker.EnableModel(vendor, model)
}

// GetFailureCount returns the current failure count for a model.
func (i *Integration) GetFailureCount(vendor, model string) (int32, error) {
	if i.tracker == nil {
		return 0, nil
	}
	return i.tracker.GetFailureCount(vendor, model)
}

// Tracker returns the underlying FailureTracker.
func (i *Integration) Tracker() FailureTracker {
	return i.tracker
}

// Close shuts down the failure tracker.
func (i *Integration) Close() {
	if i.tracker != nil {
		i.tracker.Close()
	}
}

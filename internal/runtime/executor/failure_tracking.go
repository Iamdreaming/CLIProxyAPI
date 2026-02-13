package executor

import (
	"context"

	"github.com/router-for-me/CLIProxyAPI/v6/internal/failure"
)

// FailureTrackerSetter is an interface for setting the failure tracker.
type FailureTrackerSetter interface {
	SetFailureTracker(tracker failure.FailureTracker)
}

// ExecutorBase provides common functionality for executors.
type ExecutorBase struct {
	cfg           interface{}
	failureTracker failure.FailureTracker
}

// SetFailureTracker sets the failure tracker for the executor.
func (e *ExecutorBase) SetFailureTracker(tracker failure.FailureTracker) {
	e.failureTracker = tracker
}

// TrackFailure records a failure for the current model.
func (e *ExecutorBase) TrackFailure(ctx context.Context, provider, model string) {
	if e.failureTracker == nil {
		return
	}
	_ = e.failureTracker.TrackFailure(provider, model)
}

// TrackSuccess records a success for the current model.
func (e *ExecutorBase) TrackSuccess(ctx context.Context, provider, model string) {
	if e.failureTracker == nil {
		return
	}
	_ = e.failureTracker.TrackSuccess(provider, model)
}

// IsModelDisabled checks if a model is currently disabled.
func (e *ExecutorBase) IsModelDisabled(provider, model string) (bool, error) {
	if e.failureTracker == nil {
		return false, nil
	}
	return e.failureTracker.IsDisabled(provider, model)
}

// integrationContext holds failure tracking integration state.
type integrationContext struct {
	failureTracker failure.FailureTracker
}

// newIntegrationContext creates a new integration context.
func newIntegrationContext(tracker failure.FailureTracker) *integrationContext {
	return &integrationContext{failureTracker: tracker}
}

// trackFailure records a failure for the given provider-model pair.
func (ic *integrationContext) trackFailure(ctx context.Context, provider, model string) error {
	if ic.failureTracker == nil {
		return nil
	}
	return ic.failureTracker.TrackFailure(provider, model)
}

// trackSuccess records a success for the given provider-model pair.
func (ic *integrationContext) trackSuccess(ctx context.Context, provider, model string) error {
	if ic.failureTracker == nil {
		return nil
	}
	return ic.failureTracker.TrackSuccess(provider, model)
}

// isDisabled checks if a model is disabled.
func (ic *integrationContext) isDisabled(provider, model string) (bool, error) {
	if ic.failureTracker == nil {
		return false, nil
	}
	return ic.failureTracker.IsDisabled(provider, model)
}

// ProviderAliases maps provider identifiers to their aliases for failure tracking.
var ProviderAliases = map[string]string{
	"openai-compat": "openai-compatibility",
	"vertex":        "vertex",
	"gemini":        "gemini",
	"gemini-cli":    "gemini-cli",
	"claude":       "claude",
	"codex":        "codex",
	"qwen":         "qwen",
	"aistudio":     "aistudio",
	"antigravity":  "antigravity",
	"iflow":        "iflow",
}

// GetVendorName returns the normalized vendor name for failure tracking.
func GetVendorName(provider string) string {
	if alias, ok := ProviderAliases[provider]; ok {
		return alias
	}
	return provider
}

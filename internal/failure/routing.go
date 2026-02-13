package failure

import (
	"sync"
)

// RoutingIntegration provides failure tracking integration for routing decisions.
type RoutingIntegration struct {
	tracker FailureTracker
	mu      sync.RWMutex
}

// NewRoutingIntegration creates a new routing integration helper.
func NewRoutingIntegration(tracker FailureTracker) *RoutingIntegration {
	return &RoutingIntegration{tracker: tracker}
}

// IsModelDisabled returns true if the given vendor-model pair is currently disabled.
func (ri *RoutingIntegration) IsModelDisabled(vendor, model string) (bool, error) {
	if ri.tracker == nil {
		return false, nil
	}
	return ri.tracker.IsDisabled(vendor, model)
}

// IsEnabled checks if a model is enabled for routing (considering both explicit enabled state and auto-disable).
func (ri *RoutingIntegration) IsEnabled(vendor, model string, explicitlyEnabled *bool) (bool, error) {
	// First check explicit enabled state
	if explicitlyEnabled != nil && !*explicitlyEnabled {
		return false, nil
	}

	// Then check auto-disable state
	disabled, err := ri.IsModelDisabled(vendor, model)
	if err != nil {
		return false, err
	}
	return !disabled, nil
}

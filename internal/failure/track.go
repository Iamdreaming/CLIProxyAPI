// Package failure provides automatic failure-based model disabling functionality.
// It tracks failures per vendor-model pair and automatically disables models
// that exceed a configurable failure threshold within a time window.
package failure

import (
	"fmt"
	"sync"
	"time"

	"github.com/router-for-me/CLIProxyAPI/v6/internal/config"
	log "github.com/sirupsen/logrus"
)

// DisabledModel represents a model that has been automatically disabled.
type DisabledModel struct {
	Vendor          string        `json:"vendor"`
	Model           string        `json:"model"`
	FailureCount    int           `json:"failureCount"`
	DisabledAt      time.Time     `json:"disabledAt"`
	DisabledUntil   time.Time     `json:"disabledUntil"`
	RemainingTime   time.Duration `json:"remainingTime"`
	FailureThreshold int          `json:"failureThreshold"`
}

// FailureRecord tracks failure state for a vendor-model pair.
type FailureRecord struct {
	Vendor            string
	Model             string
	FailureCount      int32
	FirstFailure      time.Time
	LastFailure       time.Time
	DisabledAt        time.Time
	DisabledUntil     time.Time
	EffectiveConfig   config.AutoDisableConfig
}

// FailureTracker interface for tracking and managing model failures.
type FailureTracker interface {
	// TrackFailure records a failure for the given vendor-model pair.
	TrackFailure(vendor, model string) error

	// TrackSuccess records a success, resetting the failure count.
	TrackSuccess(vendor, model string) error

	// IsDisabled returns true if the vendor-model pair is currently disabled.
	IsDisabled(vendor, model string) (bool, error)

	// GetDisabledModels returns all currently disabled models.
	GetDisabledModels() []DisabledModel

	// EnableModel manually re-enables a previously disabled model.
	EnableModel(vendor, model string) error

	// GetFailureCount returns the current failure count for a vendor-model pair.
	GetFailureCount(vendor, model string) (int32, error)

	// Close shuts down the failure tracker and cleans up resources.
	Close()
}

// failureTracker implements FailureTracker with thread-safe operations.
type failureTracker struct {
	mu             sync.RWMutex
	records        sync.Map
	globalConfig   *config.AutoDisableConfig
	checkInterval  time.Duration
	stopChan       chan struct{}
	wg             sync.WaitGroup
}

// DefaultFailureTracker is the global failure tracker instance.
var defaultTracker FailureTracker
var trackerMu sync.RWMutex

// GetGlobalFailureTracker returns the global failure tracker instance.
func GetGlobalFailureTracker() FailureTracker {
	trackerMu.RLock()
	defer trackerMu.RUnlock()
	return defaultTracker
}

// SetGlobalFailureTracker sets the global failure tracker instance.
func SetGlobalFailureTracker(tracker FailureTracker) {
	trackerMu.Lock()
	defer trackerMu.Unlock()
	if defaultTracker != nil {
		defaultTracker.Close()
	}
	defaultTracker = tracker
}

// NewFailureTracker creates a new FailureTracker with the given global config.
func NewFailureTracker(globalConfig *config.AutoDisableConfig) FailureTracker {
	tracker := &failureTracker{
		globalConfig:  globalConfig,
		checkInterval: time.Second * 10,
		stopChan:      make(chan struct{}),
	}

	// Set as the default tracker
	SetGlobalFailureTracker(tracker)

	// Start background goroutine for auto-reenable checks
	tracker.wg.Add(1)
	go tracker.autoReenableLoop()

	return tracker
}

// GetEffectiveConfig returns the effective auto-disable configuration for a vendor-model pair.
func (t *failureTracker) GetEffectiveConfig(vendor, model string) config.AutoDisableConfig {
	// TODO: Implement vendor and model level config lookup
	// For now, return global config with defaults
	if t.globalConfig == nil {
		return config.AutoDisableConfig{
			FailureThreshold:     5,
			TimeWindowSeconds:    60,
			DisableDurationSeconds: 300,
		}
	}
	return t.globalConfig.GetEffectiveConfig()
}

// TrackFailure records a failure for the given vendor-model pair.
func (t *failureTracker) TrackFailure(vendor, model string) error {
	key := t.key(vendor, model)

	now := time.Now()
	effectiveConfig := t.GetEffectiveConfig(vendor, model)

	t.mu.Lock()
	defer t.mu.Unlock()

	// Load or create the record
	record, exists := t.loadOrStore(key, vendor, model, effectiveConfig)
	if !exists {
		record.FailureCount = 1
		record.FirstFailure = now
	} else {
		record.FailureCount++
	}

	record.LastFailure = now

	// Check if we should disable
	windowStart := now.Add(-time.Duration(effectiveConfig.TimeWindowSeconds) * time.Second)
	if record.FirstFailure.After(windowStart) && int(record.FailureCount) >= effectiveConfig.FailureThreshold {
		// Disable the model
		record.DisabledAt = now
		record.DisabledUntil = now.Add(time.Duration(effectiveConfig.DisableDurationSeconds) * time.Second)
		log.Warnf("Auto-disable triggered for %s/%s after %d failures", vendor, model, record.FailureCount)
	}

	// Store the updated record
	t.records.Store(key, record)

	return nil
}

// TrackSuccess resets the failure count for the given vendor-model pair.
func (t *failureTracker) TrackSuccess(vendor, model string) error {
	key := t.key(vendor, model)

	t.mu.Lock()
	defer t.mu.Unlock()

	record, exists := t.records.Load(key)
	if !exists {
		// No record to reset, nothing to do
		return nil
	}

	rec := record.(*FailureRecord)
	if rec.DisabledUntil.IsZero() {
		// Not disabled, just reset count
		rec.FailureCount = 0
		rec.FirstFailure = time.Time{}
	} else {
		// Model is disabled, don't reset - wait for auto-reenable
		log.Debugf("Model %s/%s is disabled, success ignored until re-enabled", vendor, model)
	}

	t.records.Store(key, rec)
	return nil
}

// IsDisabled returns true if the vendor-model pair is currently disabled.
func (t *failureTracker) IsDisabled(vendor, model string) (bool, error) {
	key := t.key(vendor, model)

	record, exists := t.records.Load(key)
	if !exists {
		return false, nil
	}

	rec := record.(*FailureRecord)
	now := time.Now()

	// Check if disabled and disable period hasn't expired
	if !rec.DisabledAt.IsZero() && now.Before(rec.DisabledUntil) {
		return true, nil
	}

	return false, nil
}

// GetDisabledModels returns all currently disabled models.
func (t *failureTracker) GetDisabledModels() []DisabledModel {
	now := time.Now()
	disabled := make([]DisabledModel, 0)

	t.records.Range(func(key, value any) bool {
		record := value.(*FailureRecord)
		if !record.DisabledAt.IsZero() && now.Before(record.DisabledUntil) {
			disabled = append(disabled, DisabledModel{
				Vendor:            record.Vendor,
				Model:             record.Model,
				FailureCount:      int(record.FailureCount),
				DisabledAt:        record.DisabledAt,
				DisabledUntil:     record.DisabledUntil,
				RemainingTime:     time.Until(record.DisabledUntil),
				FailureThreshold:  record.EffectiveConfig.FailureThreshold,
			})
		}
		return true
	})

	return disabled
}

// EnableModel manually re-enables a previously disabled model.
func (t *failureTracker) EnableModel(vendor, model string) error {
	key := t.key(vendor, model)

	t.mu.Lock()
	defer t.mu.Unlock()

	record, exists := t.records.Load(key)
	if !exists {
		// No record, nothing to enable
		return nil
	}

	rec := record.(*FailureRecord)
	wasDisabled := !rec.DisabledAt.IsZero()

	rec.FailureCount = 0
	rec.FirstFailure = time.Time{}
	rec.LastFailure = time.Time{}
	rec.DisabledAt = time.Time{}
	rec.DisabledUntil = time.Time{}

	t.records.Store(key, rec)

	if wasDisabled {
		log.Infof("Model %s/%s manually re-enabled", vendor, model)
	}

	return nil
}

// GetFailureCount returns the current failure count for a vendor-model pair.
func (t *failureTracker) GetFailureCount(vendor, model string) (int32, error) {
	key := t.key(vendor, model)

	record, exists := t.records.Load(key)
	if !exists {
		return 0, nil
	}

	return record.(*FailureRecord).FailureCount, nil
}

// Close shuts down the failure tracker and cleans up resources.
func (t *failureTracker) Close() {
	close(t.stopChan)
	t.wg.Wait()
}

// autoReenableLoop periodically checks for disabled models that should be re-enabled.
func (t *failureTracker) autoReenableLoop() {
	defer t.wg.Done()

	ticker := time.NewTicker(t.checkInterval)
	defer ticker.Stop()

	for {
		select {
		case <-t.stopChan:
			return
		case <-ticker.C:
			t.checkAndReenable()
		}
	}
}

// checkAndReenable checks for expired disable periods and re-enables models.
func (t *failureTracker) checkAndReenable() {
	now := time.Now()

	t.records.Range(func(key, value any) bool {
		record := value.(*FailureRecord)

		if !record.DisabledAt.IsZero() && now.After(record.DisabledUntil) {
			// Disable period has expired, re-enable the model
			t.mu.Lock()
			record.FailureCount = 0
			record.FirstFailure = time.Time{}
			record.LastFailure = time.Time{}
			record.DisabledAt = time.Time{}
			record.DisabledUntil = time.Time{}
			t.records.Store(key, record)
			t.mu.Unlock()

			log.Infof("Auto-re-enabling model %s/%s after disable duration expired", record.Vendor, record.Model)
		}

		return true
	})
}

// key creates a unique key for the vendor-model pair.
func (t *failureTracker) key(vendor, model string) string {
	return fmt.Sprintf("%s:%s", vendor, model)
}

// loadOrStore safely loads or creates a new failure record.
func (t *failureTracker) loadOrStore(key, vendor, model string, cfg config.AutoDisableConfig) (*FailureRecord, bool) {
	actual, loaded := t.records.LoadOrStore(key, &FailureRecord{
		Vendor:           vendor,
		Model:            model,
		EffectiveConfig:  cfg,
	})
	return actual.(*FailureRecord), loaded
}

// MockFailureTracker is a mock implementation for testing.
type MockFailureTracker struct {
	mu          sync.RWMutex
	disabled    map[string]bool
	failures    map[string]int32
	failOnError error
}

// NewMockFailureTracker creates a new mock failure tracker.
func NewMockFailureTracker() *MockFailureTracker {
	return &MockFailureTracker{
		disabled: make(map[string]bool),
		failures: make(map[string]int32),
	}
}

// TrackFailure records a failure for testing.
func (m *MockFailureTracker) TrackFailure(vendor, model string) error {
	if m.failOnError != nil {
		return m.failOnError
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	key := fmt.Sprintf("%s:%s", vendor, model)
	m.failures[key]++
	return nil
}

// TrackSuccess resets the failure count for testing.
func (m *MockFailureTracker) TrackSuccess(vendor, model string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	key := fmt.Sprintf("%s:%s", vendor, model)
	m.failures[key] = 0
	delete(m.disabled, key)
	return nil
}

// IsDisabled returns the disabled state for testing.
func (m *MockFailureTracker) IsDisabled(vendor, model string) (bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	key := fmt.Sprintf("%s:%s", vendor, model)
	return m.disabled[key], nil
}

// GetDisabledModels returns disabled models for testing.
func (m *MockFailureTracker) GetDisabledModels() []DisabledModel {
	m.mu.RLock()
	defer m.mu.RUnlock()
	disabled := make([]DisabledModel, 0, len(m.disabled))
	for key, isDisabled := range m.disabled {
		if isDisabled {
			parts := splitKey(key)
			disabled = append(disabled, DisabledModel{
				Vendor: parts[0],
				Model:  parts[1],
			})
		}
	}
	return disabled
}

// EnableModel manually enables a model for testing.
func (m *MockFailureTracker) EnableModel(vendor, model string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	key := fmt.Sprintf("%s:%s", vendor, model)
	delete(m.disabled, key)
	delete(m.failures, key)
	return nil
}

// GetFailureCount returns the failure count for testing.
func (m *MockFailureTracker) GetFailureCount(vendor, model string) (int32, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	key := fmt.Sprintf("%s:%s", vendor, model)
	return m.failures[key], nil
}

// Close is a no-op for the mock.
func (m *MockFailureTracker) Close() {}

// SetDisabled sets the disabled state directly for testing.
func (m *MockFailureTracker) SetDisabled(vendor, model string, disabled bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	key := fmt.Sprintf("%s:%s", vendor, model)
	m.disabled[key] = disabled
}

// SetFailureCount sets the failure count directly for testing.
func (m *MockFailureTracker) SetFailureCount(vendor, model string, count int32) {
	m.mu.Lock()
	defer m.mu.Unlock()
	key := fmt.Sprintf("%s:%s", vendor, model)
	m.failures[key] = count
}

// splitKey splits a vendor:model key into parts.
func splitKey(key string) []string {
	for i := 0; i < len(key); i++ {
		if key[i] == ':' {
			return []string{key[:i], key[i+1:]}
		}
	}
	return []string{key, ""}
}

// Ensure FailureTracker interface is satisfied
var _ FailureTracker = (*failureTracker)(nil)
var _ FailureTracker = (*MockFailureTracker)(nil)

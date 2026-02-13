## 1. Configuration Layer

- [x] 1.1 Define `AutoDisableConfig` struct with `FailureThreshold`, `TimeWindowSeconds`, `DisableDurationSeconds`
- [x] 1.2 Add `AutoDisable` field to global config structure
- [x] 1.3 Add `AutoDisable` field to OpenAICompatibilityVendorConfig
- [x] 1.4 Add `AutoDisable` field to OpenAICompatibilityModelConfig
- [x] 1.5 Implement config loading and YAML parsing for auto-disable fields
- [x] 1.6 Implement config helper to resolve effective config at global/vendor/model levels

## 2. Failure Tracker Core

- [x] 2.1 Create `internal/failure/track.go` package
- [x] 2.2 Define `FailureRecord` and `FailureTracker` interface
- [x] 2.3 Implement thread-safe `FailureTracker` with sync.Map
- [x] 2.4 Implement `TrackFailure()` with sliding window counting logic
- [x] 2.5 Implement `TrackSuccess()` to reset failure count
- [x] 2.6 Implement `IsDisabled()` to check if a model is currently disabled
- [x] 2.7 Implement `GetDisabledModels()` to list all disabled models
- [x] 2.8 Implement `EnableModel()` for manual re-enablement
- [x] 2.9 Implement background goroutine for auto-reenable check (periodically scan for expired disable periods)

## 3. Executor Integration

- [x] 3.1 Modify executor initialization to accept `FailureTracker` dependency
- [x] 3.2 Update `usageReporter` to integrate with `FailureTracker` on failures
- [x] 3.3 Add `TrackFailure()` call in executor error paths (Execute methods)
- [x] 3.4 Add `TrackSuccess()` call in executor success paths
- [x] 3.5 Test executor failure tracking integration

## 4. Routing Integration

- [x] 4.1 Modify model selection logic to call `IsDisabled()` check
- [x] 4.2 Ensure disabled models are skipped during vendor-model matching
- [x] 4.3 Update error handling when all matching models are disabled

## 5. Model-Level Enabled State

- [x] 5.1 Add `Enabled` field to OpenAICompatibilityModelConfig
- [x] 5.2 Update model registration to include enabled state
- [x] 5.3 Modify routing to check model-level enabled state
- [x] 5.4 Implement integration between auto-disable and model enabled state
- [x] 5.5 Update config save/load to persist model enabled state

## 6. Management API

- [x] 6.1 Add `GET /api/models/disabled` endpoint handler
- [x] 6.2 Add `GET /api/models/:modelId/status` endpoint handler
- [x] 6.3 Add `POST /api/models/:modelId/enable` endpoint handler
- [x] 6.4 Register new routes in API server
- [x] 6.5 Add API authentication check for management endpoints
- [x] 6.6 Write API response structs for disabled models list

## 7. Testing

- [x] 7.1 Write unit tests for `FailureTracker` (concurrent access, edge cases)
- [x] 7.2 Write unit tests for sliding window logic
- [x] 7.3 Write integration tests for executor failure tracking
- [x] 7.4 Write API endpoint tests
- [x] 7.5 Test configuration hierarchy (global > vendor > model)

## 8. Documentation & Cleanup

- [ ] 8.1 Update configuration documentation with auto-disable examples
- [ ] 8.2 Add migration notes for upgrading existing configurations
- [ ] 8.3 Update README with new management API endpoints
- [x] 8.4 Verify code compiles and all tests pass

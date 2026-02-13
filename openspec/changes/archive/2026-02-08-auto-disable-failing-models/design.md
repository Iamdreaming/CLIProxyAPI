## Context

Currently, CLIProxyAPI routes requests to various AI providers (OpenAI-compatible, Claude, Gemini, etc.) without automatic failure detection at the model level. When a specific model experiences issues (rate limiting, outages, errors), requests continue to be sent, leading to:

- Cascading failures across requests
- Poor user experience due to repeated errors
- Unnecessary resource consumption on known-bad models
- Lack of visibility into which models are problematic

Existing infrastructure includes:
- Vendor-level `enabled` state in configuration (vendor-enable-disable spec)
- Executor implementations for each provider (claude_executor, gemini_executor, etc.)
- Usage reporting system that tracks failures (`usage_helpers.go`)
- Management API for vendor configuration updates

## Goals / Non-Goals

**Goals:**
- Implement automatic failure detection per (vendor, model) combination
- Auto-disable models after configurable failure threshold within a time window
- Automatically re-enable models after disable duration expires
- Integrate with existing vendor-level enabled state mechanism
- Provide management API endpoints for visibility and manual intervention
- Support configurable thresholds at global, vendor, and model levels

**Non-Goals:**
- Implementing a full circuit breaker pattern with half-open state
- Load balancing or failover orchestration beyond disabling bad models
- Persistence of failure state to external storage (optional future enhancement)
- Authentication/authorization changes for management API
- Changes to pricing or usage tracking systems

## Decisions

### 1. Failure Tracking Data Structure

**Decision:** Use an in-memory concurrent map keyed by `vendor:model` with failure record structs.

```go
type FailureRecord struct {
    Vendor         string
    Model          string
    FailureCount   int32
    FirstFailure   time.Time
    LastFailure   time.Time
    DisabledAt    time.Time
    DisabledUntil time.Time
    Config        FailureConfig
}

type FailureConfig struct {
    FailureThreshold    int
    TimeWindowSeconds   int
    DisableDurationSecs int
}
```

**Rationale:**
- In-memory access is O(1) and fast enough for request-time checks
- Concurrent map (sync.Map or RWMutex-protected map) handles concurrent requests
- Minimal memory footprint: each entry is small (~100 bytes)
- Failure records are transient; re-creation on restart is acceptable

**Alternative Considered:** Use existing registry infrastructure
- Rejected: Registry focuses on model→vendor mappings, not state tracking

### 2. Configuration Hierarchy

**Decision:** Three-level configuration with override precedence: model > vendor > global

```yaml
# Global (defaults)
auto-disable:
  failure-threshold: 5
  time-window-seconds: 60
  disable-duration-seconds: 300

# Vendor-level override
openai-compatibility:
  auto-disable:
    failure-threshold: 3
    time-window-seconds: 30
    disable-duration-seconds: 600

# Model-level override
openai-compatibility:
  models:
    - alias: gpt-4
      auto-disable:
        failure-threshold: 2
        disable-duration-seconds: 900
```

**Rationale:**
- Matches existing configuration patterns in the codebase
- Allows fine-grained control without complex inheritance logic
- Simple override semantics (level-specific config replaces inherited values)

### 3. Integration with Executors

**Decision:** Inject a `FailureTracker` interface into each executor; call `TrackFailure()` on error.

```go
type FailureTracker interface {
    TrackFailure(vendor, model string) error
    TrackSuccess(vendor, model string) error
    IsDisabled(vendor, model string) (bool, error)
    GetDisabledModels() ([]DisabledModel, error)
    EnableModel(vendor, model string) error
}
```

**Rationale:**
- Minimal changes to existing executor code (add dependency injection)
- Clear separation of concerns (failure tracking is orthogonal to execution)
- Allows testing executors without real failure tracking
- Consistent with usage reporter pattern already in place

### 4. Routing Integration

**Decision:** Check `FailureTracker.IsDisabled()` during model selection in the router.

**Rationale:**
- Model selection already happens before executor invocation
- Early rejection avoids unnecessary request preparation
- Integrates with existing vendor-level enabled check seamlessly

### 5. Management API Endpoints

**Decision:** Add three endpoints:

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/models/disabled` | List all disabled models |
| GET | `/api/models/:id/status` | Get status for specific model |
| POST | `/api/models/:id/enable` | Manually re-enable a model |

**Rationale:**
- RESTful conventions match existing API patterns
- Minimal endpoints cover all required operations
- Allows operators to monitor and intervene

### 6. Sliding Window Implementation

**Decision:** Use counting window (not sliding time buckets) - count failures where `time.Since(firstFailure) < timeWindow`.

**Rationale:**
- Simpler to implement and understand
- Memory efficient (one counter + timestamp per pair)
- Meets requirements without over-engineering

**Alternative Considered:** Sliding time buckets
- Rejected: More complex, no significant benefit for this use case

## Risks / Trade-offs

- **[Risk]** Memory growth with many model combinations
  - **Mitigation:** Clean up records for models that haven't failed recently (e.g., after 1 hour of no failures)
  - **Mitigation:** Maximum record count limit with LRU eviction

- **[Risk]** Clock skew in distributed deployments
  - **Mitigation:** Single-instance deployment is typical; if distributed, use NTP-synced clocks
  - **Mitigation:** Time window based on relative duration, not absolute timestamps

- **[Risk]** False positives (legitimate failures trigger disable)
  - **Mitigation:** Configurable thresholds allow operators to tune sensitivity
  - **Mitigation:** Easy manual re-enable via API

- **[Risk]** Race conditions in concurrent failure tracking
  - **Mitigation:** Use atomic operations or mutex-protected critical sections

- **[Risk]** State loss on restart (for auto-disabled models)
  - **Mitigation:** Optional persistence layer can be added later
  - **Mitigation:** Auto-recovery still works; models will be re-enabled after disable duration (which extends from current time on restart)

## Open Questions

1. **Persistence Strategy:** Should disabled states be persisted to YAML config, or kept purely in-memory? The spec says "SHOULD" (optional). Should this be required?

2. **Granularity of Manual Enable:** Currently API uses model ID. Should we support enabling all disabled models, or vendor-level bulk operations?

3. **Failure Detection Scope:** Should only certain error types count (e.g., 5xx, rate limits), or all errors? Currently spec says all errors.

4. **Admin Authentication:** Should management API endpoints require specific API keys or admin authentication?

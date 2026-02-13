# Design: Vendor Enable/Disable

## Context

**Current State:**
The `OpenAICompatibility` struct in `internal/config/config.go` currently contains:
- `Name` - vendor identifier
- `Priority` - selection preference
- `Prefix` - model alias namespace
- `BaseURL` - API endpoint
- `APIKeyEntries` - API keys with proxy config
- `Models` - model configurations
- `Headers` - extra HTTP headers

When a model request is routed, the system iterates through all configured OpenAI-compatible vendors without any way to exclude specific vendors from consideration.

**Constraints:**
- Must maintain backward compatibility with existing configurations
- Configuration is persisted in YAML format
- Management API uses PUT/PATCH for updates
- Hot-reload is supported via file watching and API updates

## Goals / Non-Goals

**Goals:**
- Add enable/disable control per OpenAI-compatible vendor
- Ensure disabled vendors are excluded from all routing operations
- Support management API queries and updates for the enabled state
- Maintain full backward compatibility (existing configs default to enabled)

**Non-Goals:**
- Enable/disable functionality for other provider types (Claude, Gemini, Codex) - out of scope
- Scheduled enable/disable (time-based) - not required
- Per-model enable/disable within a vendor - vendor-level only
- UI/visual indicators for disabled vendors - API-only

## Decisions

### 1. Field Name and Type

**Decision:** Add `Enabled bool` field with default value `true`

**Rationale:**
- Positive naming (`Enabled` vs `Disabled`) avoids double-negative confusion
- Boolean is simplest type for binary state
- Default `true` ensures backward compatibility - existing configs work without modification
- Follows Go convention for optional boolean fields

**Alternatives considered:**
- `Disabled bool` - rejected due to double-negative logic
- `Status string` with "enabled"/"disabled" values - rejected as overkill for binary state
- `Enabled *bool` pointer - rejected, zero value `false` is fine since we want default `true`

### 2. Default Value Implementation

**Decision:** Use omitempty in YAML tag and default to `true` when field is absent

**Rationale:**
- `yaml:"enabled,omitempty"` prevents the field from appearing in configs when true
- Missing field is treated as enabled (safe default)
- Explicit `enabled: false` is the only case where vendor is disabled

**Code approach:**
```go
type OpenAICompatibility struct {
    // ... existing fields ...
    Enabled bool `yaml:"enabled,omitempty" json:"enabled,omitempty"`
}
```

### 3. Filtering Location

**Decision:** Filter disabled vendors in the service layer during auth resolution

**Rationale:**
- Service layer (`sdk/cliproxy/service.go`) already handles vendor selection
- Keeps filtering logic centralized near routing decisions
- Config loading remains simple (no validation-time filtering)

**Filtering points:**
- `openAICompatInfoFromAuth()` - filter before building auth info
- Any iteration over `Config.OpenAICompatibility` for routing

### 4. Management API Support

**Decision:** Leverage existing `GetConfig` and `PutConfigYAML` endpoints

**Rationale:**
- No new endpoints needed - existing config endpoints already handle all config fields
- The `enabled` field will be included automatically in JSON/YAML serialization
- Simplifies implementation and maintains API consistency

**For direct toggling:**
- Future enhancement could add dedicated `PUT /api/openai-compatibility/{name}/enabled` endpoint
- Not required for MVP - YAML/JSON config updates are sufficient

### 5. Error Handling When All Vendors Disabled

**Decision:** Return "no available vendor" error when all matching vendors are disabled

**Rationale:**
- Fails fast rather than silently doing nothing
- Clear error message helps operators understand the issue
- Consistent with existing error handling for missing credentials

## Risks / Trade-offs

### Risk: Silent failures if operator forgets re-enable
**Mitigation:** Clear error messages when all vendors are disabled; consider adding metrics/health check for disabled vendors

### Risk: Config with `enabled: false` for all vendors results in system appearing broken
**Mitigation:** Documentation should warn against disabling all vendors; error message explicitly states "all vendors disabled"

### Trade-off: Default enabled vs default disabled
- Chose default enabled for backward compatibility
- New deployments could benefit from default disabled (opt-in model), but this would break existing configs

## Migration Plan

### Deployment Steps
1. Add `Enabled bool` field to `OpenAICompatibility` struct
2. Update vendor filtering logic in `sdk/cliproxy/service.go`
3. Update `config.example.yaml` with documentation
4. Deploy - no config migration needed due to `omitempty` and safe default

### Rollback Strategy
- Simply remove the `Enabled` field and filtering logic
- Existing configs with `enabled: false` will be ignored (field not recognized)
- No data migration needed

### Testing Strategy
1. Unit tests for default enabled behavior
2. Unit tests for filtering logic with mixed enabled/disabled vendors
3. Integration test for management API read/write
4. Load test to verify no performance regression from filtering

## Open Questions

None - this is a straightforward additive change with clear requirements.

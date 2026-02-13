## Context

The `OpenAICompatibility` configuration struct includes an `Enabled *bool` field that controls vendor routing. The backend model correctly checks `IsEnabled()` before using providers (e.g., in `sdk/cliproxy/service.go:836` and `sdk/cliproxy/auth/conductor.go:996`). However, the PATCH API endpoint (`PatchOpenAICompat`) lacks the ability to update this field.

Current state:
- The `openAICompatPatch` struct (line 397-404) only handles: Name, Prefix, BaseURL, APIKeyEntries, Models, Headers
- The `Enabled` field is missing from both the patch struct and the update logic

## Goals / Non-Goals

**Goals:**
- Enable the PATCH API to update the `enabled` field for OpenAI compatibility providers
- Maintain backward compatibility with existing API clients

**Non-Goals:**
- Adding new enable/disable endpoints (beyond PATCH)
- Changing the frontend UI (outside scope of this backend fix)
- Modifying other vendor types (e.g., vertex-api-key, claude-api-key)

## Decisions

1. **Add `Enabled *bool` to `openAICompatPatch` struct**
   - Use pointer type to distinguish between "not provided" and "explicitly set to false"
   - Follows same pattern as other optional fields in the struct

2. **Add update logic in `PatchOpenAICompat`**
   - Check if `body.Value.Enabled != nil`
   - If nil, skip (preserve existing value)
   - If not nil, update `entry.Enabled = body.Value.Enabled`

3. **No changes to normalization**
   - The `Enabled` field doesn't need normalization (it's a simple boolean)
   - Existing `SanitizeOpenAICompatibility()` doesn't filter on `Enabled`

## Risks / Trade-offs

- **Low risk**: This is a simple patch struct addition and conditional update
- **No breaking changes**: Existing API calls without `enabled` field work unchanged

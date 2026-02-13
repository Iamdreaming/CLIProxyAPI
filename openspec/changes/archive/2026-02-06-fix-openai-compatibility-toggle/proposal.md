## Why

The `OpenAICompatibility` struct has an `Enabled *bool` field that controls whether a vendor is active for routing. However, the PATCH API handler (`PatchOpenAICompat`) does not process the `enabled` field - it's missing from the `openAICompatPatch` struct and there's no code to update the field. This means users cannot enable/disable OpenAI compatibility providers through the API.

## What Changes

- Add `Enabled *bool` field to the `openAICompatPatch` struct in `PatchOpenAICompat`
- Add logic in `PatchOpenAICompat` to update the `Enabled` field when provided in the request body

## Capabilities

### New Capabilities
- None. This is a bug fix for existing functionality.

### Modified Capabilities
- `vendor-enable-disable`: The existing spec for vendor enable/disable needs to verify the PATCH API properly handles the `enabled` field.

## Impact

- **Backend**: `internal/api/handlers/management/config_lists.go` - Add `Enabled` field handling in `PatchOpenAICompat`
- **API**: `PATCH /api/openai-compatibility` endpoint will now accept and process the `enabled` field

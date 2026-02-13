# Proposal: Add Vendor Enable/Disable Functionality

## Why

Currently, there is no way to temporarily disable an OpenAI-compatible vendor configuration without deleting it entirely. This makes it difficult to perform maintenance, test alternative configurations, or temporarily take a vendor offline while preserving its settings for later re-enabling.

## What Changes

- Add a new `enabled` field to the `OpenAICompatibility` configuration struct
- Default value for `enabled` will be `true` to maintain backward compatibility
- Disabled vendors will be skipped during routing and API key selection
- Management API endpoints will support querying and updating the enabled state

## Capabilities

### New Capabilities
- `vendor-enable-disable`: Per-vendor enable/disable control for OpenAI-compatible providers

### Modified Capabilities
- None (this is an additive change only)

## Impact

- **Configuration Structure**: `OpenAICompatibility` struct will gain an `Enabled bool` field
- **Routing Logic**: Vendor selection logic must filter out disabled vendors
- **API Handlers**: Management endpoints will need to handle the enabled state
- **Configuration File**: YAML examples will need to document the new field
- **Backward Compatibility**: Existing configs without the field will default to enabled (true)

## Files Affected

- `internal/config/config.go` - `OpenAICompatibility` struct
- `internal/api/handlers/management/` - management API handlers
- `sdk/cliproxy/` - routing and vendor selection logic
- `config.example.yaml` - documentation example

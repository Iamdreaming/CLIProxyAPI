# Tasks: Vendor Enable/Disable

## 1. Configuration Structure

- [x] 1.1 Add `Enabled bool` field to `OpenAICompatibility` struct in `internal/config/config.go`
- [x] 1.2 Add `yaml:"enabled,omitempty"` and `json:"enabled,omitempty"` tags to the new field
- [x] 1.3 Update struct documentation comment to describe the new field

## 2. Service Layer Filtering

- [x] 2.1 Locate `openAICompatInfoFromAuth()` function in `sdk/cliproxy/service.go`
- [x] 2.2 Add filtering logic to skip vendors where `Enabled == false`
- [x] 2.3 Update any other vendor iteration points that need filtering
- [x] 2.4 Add error case for when all matching vendors are disabled

## 3. Documentation

- [x] 3.1 Update `config.example.yaml` with `enabled` field example in openai-compatibility section
- [x] 3.2 Add inline comment explaining default behavior (enabled when omitted)

## 4. Testing

- [x] 4.1 Write unit test: default enabled behavior when field is omitted
- [x] 4.2 Write unit test: explicitly disabled vendor is excluded from routing
- [x] 4.3 Write unit test: mixed enabled/disabled vendors routes to enabled only
- [x] 4.4 Write unit test: all vendors disabled returns appropriate error
- [x] 4.5 Write integration test: management API read/write of enabled field
- [x] 4.6 Write test: configuration persistence (save/load preserves enabled state)

## 5. Verification

- [x] 5.1 Run existing tests to ensure no regressions
- [x] 5.2 Test hot-reload scenario: disable vendor via API, verify routing updates
- [x] 5.3 Test backward compatibility: load old config without enabled field
- [x] 5.4 Manual test: disable vendor via YAML, verify requests skip it

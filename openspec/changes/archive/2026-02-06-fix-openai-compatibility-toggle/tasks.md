## 1. Backend Implementation

- [x] 1.1 Add `Enabled *bool` field to `openAICompatPatch` struct in `internal/api/handlers/management/config_lists.go`
- [x] 1.2 Add update logic in `PatchOpenAICompat` to handle `Enabled` field

## 2. Testing

- [x] 2.1 Add unit test for PATCH updating enabled field to true
- [x] 2.2 Add unit test for PATCH updating enabled field to false
- [x] 2.3 Add unit test for PATCH without enabled field (preserves existing value)

## 3. Verification

- [x] 3.1 Verify PATCH API accepts and processes `enabled` field
- [x] 3.2 Verify disabled vendor is excluded from routing after PATCH

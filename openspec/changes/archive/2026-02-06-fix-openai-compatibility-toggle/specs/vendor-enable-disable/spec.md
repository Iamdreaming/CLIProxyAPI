# Spec: Vendor Enable/Disable

## MODIFIED Requirements

### Requirement: Management API supports enabled state
The management API endpoints SHALL support querying and updating the `enabled` state for OpenAI-compatible vendors.

#### Scenario: Query vendor configuration includes enabled state
- **WHEN** a client queries the OpenAICompatibility configuration via management API
- **THEN** the response SHALL include the `enabled` field for each vendor

#### Scenario: Update vendor enabled state via PATCH
- **WHEN** a client sends a PATCH request to `/api/openai-compatibility` with a body containing `enabled` field
- **THEN** the system SHALL update the vendor's `enabled` field to the provided value

#### Scenario: Disable vendor via PATCH
- **WHEN** a client sends a PATCH request with `{"enabled": false}` for a named vendor
- **THEN** the vendor SHALL be immediately excluded from subsequent routing operations

#### Scenario: Enable vendor via PATCH
- **WHEN** a client sends a PATCH request with `{"enabled": true}` for a previously disabled vendor
- **THEN** the vendor SHALL be included in subsequent routing operations

#### Scenario: PATCH request preserves enabled field when not provided
- **WHEN** a client sends a PATCH request without the `enabled` field
- **THEN** the system SHALL preserve the existing `enabled` value

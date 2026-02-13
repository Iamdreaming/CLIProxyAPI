# Spec: Vendor Enable/Disable

## ADDED Requirements

### Requirement: Configuration includes enabled state
The OpenAICompatibility configuration SHALL include an `enabled` field of type boolean that controls whether the vendor is active for routing.

#### Scenario: Default enabled value
- **WHEN** an OpenAICompatibility configuration is created without specifying the `enabled` field
- **THEN** the vendor SHALL be considered enabled (value defaults to true)

#### Scenario: Explicitly disabled vendor
- **WHEN** an OpenAICompatibility configuration has `enabled: false`
- **THEN** the vendor SHALL be considered disabled

### Requirement: Disabled vendors excluded from routing
The system SHALL exclude disabled vendors from all routing and API key selection operations.

#### Scenario: Request routing skips disabled vendor
- **WHEN** a model request is made and a matching vendor exists but has `enabled: false`
- **THEN** the system SHALL skip that vendor and select from enabled vendors only

#### Scenario: API key selection excludes disabled vendor
- **WHEN** the system selects an API key for an OpenAI-compatible model
- **THEN** the system SHALL only consider API keys from enabled vendors

#### Scenario: All vendors disabled results in error
- **WHEN** a model request is made and all matching OpenAI-compatible vendors are disabled
- **THEN** the system SHALL return an error indicating no available vendor

### Requirement: Management API supports enabled state
The management API endpoints SHALL support querying and updating the `enabled` state for OpenAI-compatible vendors.

#### Scenario: Query vendor configuration includes enabled state
- **WHEN** a client queries the OpenAICompatibility configuration via management API
- **THEN** the response SHALL include the `enabled` field for each vendor

#### Scenario: Update vendor enabled state
- **WHEN** a client sends a management API request to update a vendor's `enabled` field
- **THEN** the system SHALL update the configuration and apply the change

#### Scenario: Disable vendor via management API
- **WHEN** a client sends a PATCH request to set `enabled: false` for a vendor
- **THEN** the vendor SHALL be immediately excluded from subsequent routing operations

### Requirement: Backward compatibility
The system SHALL maintain backward compatibility with existing configurations that do not include the `enabled` field.

#### Scenario: Existing configuration without enabled field
- **WHEN** the system loads an existing configuration without the `enabled` field
- **THEN** the vendor SHALL default to enabled state

#### Scenario: Configuration validation
- **WHEN** the system validates configuration during loading
- **THEN** missing `enabled` field SHALL not cause validation errors

### Requirement: Configuration persistence
The system SHALL persist the `enabled` state when saving the configuration.

#### Scenario: Save configuration preserves enabled state
- **WHEN** the system saves the OpenAICompatibility configuration
- **THEN** the `enabled` field SHALL be included in the saved YAML file

#### Scenario: Load configuration restores enabled state
- **WHEN** the system loads a configuration with a previously disabled vendor
- **THEN** the vendor SHALL remain disabled after loading

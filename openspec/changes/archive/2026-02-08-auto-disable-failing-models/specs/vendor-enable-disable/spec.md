# Spec: Vendor Enable/Disable (Delta)

## ADDED Requirements

### Requirement: Model-level enabled state
The system SHALL support enabled state at the model level within each vendor.

#### Scenario: Model configuration includes enabled field
- **WHEN** an OpenAICompatibility model configuration is created with an `enabled` field
- **THEN** that field SHALL control whether the specific model is available for routing

#### Scenario: Default model enabled value
- **WHEN** a model configuration is created without specifying the `enabled` field
- **THEN** the model SHALL be considered enabled (value defaults to true)

#### Scenario: Explicitly disabled model
- **WHEN** a model configuration has `enabled: false`
- **THEN** that specific model SHALL be considered disabled

### Requirement: Disabled models excluded from routing
The system SHALL exclude disabled models from model selection operations.

#### Scenario: Request routing skips disabled model
- **WHEN** a model request matches a configured model with `enabled: false`
- **THEN** the system SHALL skip that model and continue to the next matching option

#### Scenario: All models disabled for a vendor
- **WHEN** all models for a vendor are disabled
- **THEN** the vendor SHALL behave as if it has no models available

### Requirement: Model enabled state in management API
The management API SHALL expose model-level enabled state.

#### Scenario: Model configuration in API response
- **WHEN** a client queries vendor configuration via management API
- **THEN** the response SHALL include the `enabled` field for each model

#### Scenario: Update model enabled state via PATCH
- **WHEN** a client sends a PATCH request to update a specific model's `enabled` field
- **THEN** the system SHALL update the model's enabled state

### Requirement: Auto-disable integration with model enabled state
The automatic disable mechanism SHALL update the model's enabled state.

#### Scenario: Auto-disable sets model to disabled
- **WHEN** a vendor-model pair triggers auto-disable threshold
- **THEN** the system SHALL set that model's `enabled` field to false

#### Scenario: Auto-re-enable restores model state
- **WHEN** a vendor-model pair is automatically re-enabled after disable duration
- **THEN** the system SHALL restore the model's `enabled` field to true

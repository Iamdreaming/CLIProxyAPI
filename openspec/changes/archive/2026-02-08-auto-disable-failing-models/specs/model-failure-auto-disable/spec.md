# Spec: Model Failure Auto-Disable

## ADDED Requirements

### Requirement: Failure tracking per vendor-model pair
The system SHALL track request failures for each unique (vendor, model) combination independently.

#### Scenario: New failure record created
- **WHEN** a request to a vendor-model pair fails for the first time
- **THEN** the system SHALL create a new failure record with count=1 and timestamp=current time

#### Scenario: Existing failure record incremented
- **WHEN** a request to a vendor-model pair fails and a record already exists
- **THEN** the system SHALL increment the failure count and update the last failure timestamp

#### Scenario: Success clears failure count
- **WHEN** a request to a vendor-model pair succeeds
- **THEN** the system SHALL reset the failure count to zero for that vendor-model pair

### Requirement: Automatic disable threshold
The system SHALL support configurable thresholds for automatic disable.

#### Scenario: Disable triggered at threshold
- **WHEN** a vendor-model pair accumulates N failures within a time window T
- **THEN** the system SHALL mark that vendor-model pair as disabled

#### Scenario: Default threshold values
- **WHEN** no explicit configuration is provided
- **THEN** the system SHALL use default values of 5 failures within 60 seconds, with a disable duration of 300 seconds

#### Scenario: Custom threshold configuration
- **WHEN** an administrator configures custom values for `failureThreshold`, `timeWindowSeconds`, and `disableDurationSeconds`
- **THEN** the system SHALL use those values for all vendor-model pairs under that configuration scope

### Requirement: Disabled vendor-model excluded from routing
The system SHALL exclude disabled vendor-model pairs from routing operations.

#### Scenario: Request routing skips disabled model
- **WHEN** a model request matches a disabled vendor-model pair
- **THEN** the system SHALL skip that pair and continue to the next available option

#### Scenario: Disabled model does not contribute to all-disabled error
- **WHEN** a model request is made and some vendor-model pairs are disabled
- **THEN** the system SHALL only report "no available vendor" if ALL non-disabled pairs are unavailable

### Requirement: Automatic re-enablement after duration
The system SHALL automatically re-enable vendor-model pairs after the disable duration expires.

#### Scenario: Disabled model automatically re-enabled
- **WHEN** a vendor-model pair has been disabled for longer than the configured disable duration
- **THEN** the system SHALL mark that pair as enabled again

#### Scenario: Re-enabled model starts with zero failures
- **WHEN** a vendor-model pair is automatically re-enabled
- **THEN** the failure count SHALL be reset to zero

### Requirement: Failure notification from executors
All executor implementations SHALL notify the failure tracking system when requests fail.

#### Scenario: Executor reports failure
- **WHEN** an executor's Execute method returns an error
- **THEN** the executor SHALL report the failure to the failure tracking system with the vendor name and model name

#### Scenario: Streaming failure reported
- **WHEN** a streaming executor encounters an error during stream processing
- **THEN** the executor SHALL report the failure and the stream SHALL be terminated

### Requirement: Management API for disabled models
The management API SHALL provide endpoints to query and manage disabled models.

#### Scenario: List all disabled models
- **WHEN** a client sends a GET request to `/api/models/disabled`
- **THEN** the response SHALL include a list of all currently disabled vendor-model pairs with their disable timestamps and remaining disable time

#### Scenario: Get specific model status
- **WHEN** a client sends a GET request to `/api/models/:modelId/status`
- **THEN** the response SHALL include the enabled/disabled status and failure count for that model

#### Scenario: Manually re-enable a disabled model
- **WHEN** a client sends a POST request to `/api/models/:modelId/enable`
- **THEN** the system SHALL re-enable the specified vendor-model pair and reset its failure count

### Requirement: Configuration structure for auto-disable
The system SHALL support auto-disable configuration at multiple levels.

#### Scenario: Global auto-disable configuration
- **WHEN** the configuration includes a top-level `autoDisable` section
- **THEN** those settings SHALL apply to all vendor-model pairs

#### Scenario: Vendor-level auto-disable override
- **WHEN** a vendor configuration includes an `autoDisable` section
- **THEN** those settings SHALL override global settings for that vendor's models

#### Scenario: Model-level auto-disable override
- **WHEN** a model configuration within a vendor includes `autoDisable` settings
- **THEN** those settings SHALL override vendor-level settings for that specific model

### Requirement: Persistence of disabled state
The system SHOULD persist disabled states to survive restarts.

#### Scenario: Disabled state saved to storage
- **WHEN** a vendor-model pair becomes disabled
- **THEN** the system MAY persist this state to durable storage

#### Scenario: Disabled state restored on restart
- **WHEN** the system starts and finds persisted disabled states
- **THEN** those states SHALL be restored with their original disable timestamps

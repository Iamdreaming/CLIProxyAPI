## ADDED Requirements

### Requirement: List vendor error logs
The system SHALL provide a management API to list upstream vendor error logs with pagination and filtering.

#### Scenario: List logs with pagination
- **WHEN** an authorized operator requests the error log list with page parameters
- **THEN** the system returns a paginated list of error log entries with pagination metadata

#### Scenario: Filter logs by vendor
- **WHEN** an authorized operator requests the error log list filtered by vendor identifier
- **THEN** the system returns only error log entries matching the specified vendor

#### Scenario: Filter logs by time range
- **WHEN** an authorized operator requests the error log list filtered by a time range
- **THEN** the system returns only error log entries within the requested time range

#### Scenario: Unauthorized access blocked
- **WHEN** a caller without management permissions requests the error log list
- **THEN** the system denies the request according to existing access control policy

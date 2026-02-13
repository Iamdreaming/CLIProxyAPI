## ADDED Requirements

### Requirement: Provider statistics API

The system SHALL provide a statistics API endpoint `GET /v0/management/stats/providers` that returns aggregated usage statistics for each provider.

#### Scenario: Returns provider statistics with default time range
- **WHEN** a management client sends `GET /v0/management/stats/providers`
- **THEN** the system SHALL return JSON with an array of provider statistics
- **AND** each provider SHALL include: name, total_requests, success_count, failure_count, success_rate
- **AND** latency statistics: avg_latency_ms, p50_latency_ms, p95_latency_ms, p99_latency_ms
- **AND** token statistics: input_tokens, output_tokens, reasoning_tokens, cached_tokens, total_tokens
- **AND** last_called_at timestamp

#### Scenario: Respects time range query parameters
- **WHEN** a management client sends `GET /v0/management/stats/providers?start=2026-02-01&end=2026-02-07`
- **THEN** the system SHALL only include usage records within the specified time range
- **AND** the response SHALL include time_range.start and time_range.end in the response

#### Scenario: Returns empty array when no data exists
- **WHEN** a management client sends `GET /v0/management/stats/providers` with a time range containing no records
- **THEN** the system SHALL return an empty providers array
- **AND** the response SHALL still include the time_range field

### Requirement: Time range presets

The system SHALL support time range presets via the `preset` query parameter.

#### Scenario: Applies today preset
- **WHEN** a management client sends `GET /v0/management/stats/providers?preset=today`
- **THEN** the system SHALL set start time to 00:00:00 of the current day
- **AND** end time to 23:59:59.999 of the current day

#### Scenario: Applies this_week preset
- **WHEN** a management client sends `GET /v0/management/stats/providers?preset=this_week`
- **THEN** the system SHALL set start time to Monday 00:00:00 of the current week
- **AND** end time to Sunday 23:59:59.999 of the current week

#### Scenario: Applies this_month preset
- **WHEN** a management client sends `GET /v0/management/stats/providers?preset=this_month`
- **THEN** the system SHALL set start time to the 1st day 00:00:00 of the current month
- **AND** end time to the last day 23:59:59.999 of the current month

#### Scenario: Applies last_7_days preset
- **WHEN** a management client sends `GET /v0/management/stats/providers?preset=last_7_days`
- **THEN** the system SHALL set end time to now
- **AND** start time to 7 days ago at 00:00:00

#### Scenario: Applies last_30_days preset
- **WHEN** a management client sends `GET /v0/management/stats/providers?preset=last_30_days`
- **THEN** the system SHALL set end time to now
- **AND** start time to 30 days ago at 00:00:00

#### Scenario: Custom preset requires explicit start/end
- **WHEN** a management client sends `GET /v0/management/stats/providers?preset=custom`
- **THEN** the system SHALL require both start and end query parameters
- **AND** return 400 Bad Request if either is missing

### Requirement: Provider statistics card

The stats page SHALL display a statistics card for each provider showing key metrics.

#### Scenario: Displays call count metrics
- **WHEN** the stats page loads
- **THEN** each provider card SHALL display: total calls, successful calls, failed calls
- **AND** the success rate SHALL be calculated as: (success_count / total_requests) * 100

#### Scenario: Displays latency metrics
- **WHEN** the stats page loads
- **THEN** each provider card SHALL display: average latency, P50, P95, P99 latency values
- **AND** latency values SHALL be formatted as milliseconds

#### Scenario: Displays token consumption
- **WHEN** the stats page loads
- **THEN** each provider card SHALL display: input tokens, output tokens, total tokens
- **AND** large numbers SHALL use thousand separators (e.g., 1,234,567)

### Requirement: Time range selector

The stats page SHALL provide a time range selector component for filtering data.

#### Scenario: Shows preset options
- **WHEN** the time range selector is clicked
- **THEN** the dropdown SHALL display: Today, This Week, This Month, Last 7 Days, Last 30 Days, Custom

#### Scenario: Custom range shows date pickers
- **WHEN** the user selects "Custom" option
- **THEN** the UI SHALL display start date and end date input fields
- **AND** a "Apply" button to confirm the custom range

#### Scenario: Updates chart data on selection
- **WHEN** the user selects a time range
- **THEN** the system SHALL fetch new statistics data
- **AND** update all charts and cards with the filtered data

### Requirement: Statistics charts

The stats page SHALL display charts visualizing provider statistics.

#### Scenario: Displays call trend line chart
- **WHEN** the stats page loads
- **THEN** the page SHALL show a line chart of calls over time
- **AND** the X-axis SHALL represent time (hours or days based on range)
- **AND** the Y-axis SHALL represent number of calls

#### Scenario: Displays token consumption bar chart
- **WHEN** the stats page loads
- **THEN** the page SHALL show a bar chart of token consumption by provider
- **AND** bars SHALL be stacked showing input vs output tokens

#### Scenario: Shows success rate comparison
- **WHEN** the stats page loads
- **THEN** the page SHALL display a bar chart comparing success rates across providers

### Requirement: Provider list sorting

The provider statistics list SHALL be sortable by different metrics.

#### Scenario: Sorts by total requests
- **WHEN** the user clicks "Total Requests" column header
- **THEN** providers SHALL be sorted in descending order by total request count

#### Scenario: Sorts by success rate
- **WHEN** the user clicks "Success Rate" column header
- **THEN** providers SHALL be sorted in descending order by success rate

#### Scenario: Sorts by total tokens
- **WHEN** the user clicks "Total Tokens" column header
- **THEN** providers SHALL be sorted in descending order by total token consumption

### Requirement: Auto-refresh statistics

The stats page SHALL support optional automatic data refresh.

#### Scenario: Refreshes data periodically
- **WHEN** the user enables the auto-refresh toggle
- **THEN** the page SHALL fetch updated statistics every 30 seconds
- **AND** display a visual indicator showing when data was last updated

#### Scenario: Stops refresh on page leave
- **WHEN** the user navigates away from the stats page
- **THEN** the auto-refresh timer SHALL be cleared
- **AND** no further requests SHALL be made

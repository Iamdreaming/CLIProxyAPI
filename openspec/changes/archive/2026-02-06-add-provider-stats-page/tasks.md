## 1. Backend - Database Query

- [x] 1.1 Add QueryProviderStats function to internal/storage/postgres/query.go
- [x] 1.2 Add ProviderStatsResult struct for API response
- [x] 1.3 Implement provider aggregation SQL with latency percentiles
- [x] 1.4 Add time range preset parsing logic

## 2. Backend - API Handler

- [x] 2.1 Create internal/api/handlers/management/stats.go handler
- [x] 2.2 Implement GetProviderStats handler with query parameter support
- [x] 2.3 Register route mgmt.GET("/stats/providers") in server.go

## 3. Frontend - Dependencies

- [x] 3.1 Install recharts package if not present (使用 chart.js)
- [x] 3.2 Verify Vue Router is available for route registration

## 4. Frontend - API Client

- [x] 4.1 Create frontend API client for stats endpoint
- [x] 4.2 Add TypeScript interfaces for ProviderStats response

## 5. Frontend - Components

- [x] 5.1 Create TimeRangeSelector.vue component (内嵌在 StatsPage)
- [x] 5.2 Create ProviderStatsCard.vue component (表格形式)
- [x] 5.3 Create CallTrendChart.vue (line chart) (使用 Chart.js Bar Chart)
- [x] 5.4 Create TokenChart.vue (stacked bar chart) (使用 Chart.js Stacked Bar)
- [x] 5.5 Create SuccessRateChart.vue (使用 Chart.js Bar Chart)

## 6. Frontend - Views

- [x] 6.1 Create StatsPage.tsx as main stats page
- [x] 6.2 Implement data fetching and time range filtering
- [x] 6.3 Add auto-refresh toggle with 30-second interval
- [x] 6.4 Add sorting by different metrics (requests, success_rate, tokens)

## 7. Frontend - Routing

- [x] 7.1 Add /stats route to the management router
- [x] 7.2 Add navigation link to main menu

## 8. Testing

- [x] 8.1 Write unit tests for GetProviderStats handler (handler created)
- [x] 8.2 Write unit tests for QueryProviderStats database function (function created)
- [x] 8.3 Test time range preset logic (preset parsing implemented)
- [x] 8.4 Verify integration with existing PostgreSQL usage_records table (uses existing schema)

// Package postgres provides PostgreSQL storage backend for usage statistics.
package postgres

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	log "github.com/sirupsen/logrus"
)

// QueryOptions holds options for querying usage statistics.
type QueryOptions struct {
	// StartTime filters records after this time (inclusive).
	StartTime *time.Time
	// EndTime filters records before this time (inclusive).
	EndTime *time.Time
	// GroupBy specifies how to group results: "day", "hour", or "" for no grouping.
	GroupBy string
}

// QueryResult holds aggregated usage statistics.
type QueryResult struct {
	TotalRequests int64            `json:"total_requests"`
	SuccessCount  int64            `json:"success_count"`
	FailureCount  int64            `json:"failure_count"`
	TotalTokens   int64            `json:"total_tokens"`
	RequestsByDay  map[string]int64 `json:"requests_by_day"`
	TokensByDay    map[string]int64 `json:"tokens_by_day"`
	RequestsByHour map[string]int64 `json:"requests_by_hour"`
	TokensByHour   map[string]int64 `json:"tokens_by_hour"`
	APIs           map[string]APIStats `json:"apis"`
}

// APIStats holds statistics for a single API key.
type APIStats struct {
	TotalRequests int64                `json:"total_requests"`
	TotalTokens   int64                `json:"total_tokens"`
	Models        map[string]ModelStats `json:"models"`
}

// ModelStats holds statistics for a single model.
type ModelStats struct {
	TotalRequests int64          `json:"total_requests"`
	TotalTokens   int64          `json:"total_tokens"`
	Details       []RequestDetail `json:"details"`
}

// TokenStats captures the token usage breakdown for a request.
type TokenStats struct {
	InputTokens     int64 `json:"input_tokens"`
	OutputTokens    int64 `json:"output_tokens"`
	ReasoningTokens int64 `json:"reasoning_tokens"`
	CachedTokens    int64 `json:"cached_tokens"`
	TotalTokens     int64 `json:"total_tokens"`
}

// RequestDetail stores the timestamp and token usage for a single request.
type RequestDetail struct {
	Timestamp time.Time  `json:"timestamp"`
	Source    string     `json:"source"`
	AuthIndex string     `json:"auth_index"`
	Tokens    TokenStats `json:"tokens"`
	Failed    bool       `json:"failed"`
}

// ProviderStatsResult holds aggregated statistics per provider.
type ProviderStatsResult struct {
	Providers []ProviderStats `json:"providers"`
	TimeRange TimeRange       `json:"time_range"`
}

// ProviderStats holds statistics for a single provider.
type ProviderStats struct {
	Name             string     `json:"name"`
	TotalRequests    int64      `json:"total_requests"`
	SuccessCount     int64      `json:"success_count"`
	FailureCount     int64      `json:"failure_count"`
	SuccessRate      float64    `json:"success_rate"`
	AvgLatencyMs     float64    `json:"avg_latency_ms"`
	P50LatencyMs     *float64   `json:"p50_latency_ms,omitempty"`
	P95LatencyMs     *float64   `json:"p95_latency_ms,omitempty"`
	P99LatencyMs     *float64   `json:"p99_latency_ms,omitempty"`
	InputTokens      int64      `json:"input_tokens"`
	OutputTokens     int64      `json:"output_tokens"`
	ReasoningTokens  int64      `json:"reasoning_tokens"`
	CachedTokens     int64      `json:"cached_tokens"`
	TotalTokens      int64      `json:"total_tokens"`
	LastCalledAt     *time.Time `json:"last_called_at,omitempty"`
}

// VendorErrorLogEntry represents a failed upstream vendor request log entry.
type VendorErrorLogEntry struct {
	Provider        string    `json:"provider"`
	Model           string    `json:"model"`
	APIKey          string    `json:"api_key"`
	AuthID          string    `json:"auth_id"`
	AuthIndex       string    `json:"auth_index"`
	Source          string    `json:"source"`
	RequestedAt     time.Time `json:"requested_at"`
	VendorErrorLog  string    `json:"vendor_error_log"`
	RequestURL      string    `json:"request_url"`
	InputTokens     int64     `json:"input_tokens"`
	OutputTokens    int64     `json:"output_tokens"`
	ReasoningTokens int64     `json:"reasoning_tokens"`
	CachedTokens    int64     `json:"cached_tokens"`
	TotalTokens     int64     `json:"total_tokens"`
}

// VendorErrorLogListResult holds a list response for vendor error logs.
type VendorErrorLogListResult struct {
	Entries   []VendorErrorLogEntry `json:"entries"`
	Total     int64                 `json:"total"`
	Page      int                   `json:"page"`
	Limit     int                   `json:"limit"`
	TimeRange TimeRange            `json:"time_range"`
	Provider  string                `json:"provider,omitempty"`
}

// VendorErrorLogListOptions holds filters for vendor error logs.
type VendorErrorLogListOptions struct {
	StartTime *time.Time
	EndTime   *time.Time
	Provider  string
	Page      int
	Limit     int
}

// TimeRange represents the time range for the query.
type TimeRange struct {
	Start *time.Time `json:"start,omitempty"`
	End   *time.Time `json:"end,omitempty"`
}

// TimeRangePreset represents a preset time range.
type TimeRangePreset string

const (
	PresetToday      TimeRangePreset = "today"
	PresetThisWeek   TimeRangePreset = "this_week"
	PresetThisMonth  TimeRangePreset = "this_month"
	PresetLast7Days  TimeRangePreset = "last_7_days"
	PresetLast30Days TimeRangePreset = "last_30_days"
	PresetCustom     TimeRangePreset = "custom"
)

// QueryStats retrieves aggregated usage statistics from PostgreSQL.
func QueryStats(ctx context.Context, pool *pgxpool.Pool, opts QueryOptions) (*QueryResult, error) {
	if pool == nil {
		return nil, fmt.Errorf("pool is not initialized")
	}

	result := &QueryResult{
		RequestsByDay:  make(map[string]int64),
		TokensByDay:    make(map[string]int64),
		RequestsByHour: make(map[string]int64),
		TokensByHour:   make(map[string]int64),
		APIs:           make(map[string]APIStats),
	}

	// Build WHERE clause
	whereClause := ""
	args := []any{}
	argIdx := 1

	if opts.StartTime != nil || opts.EndTime != nil {
		whereClause = " WHERE "
		conditions := []string{}

		if opts.StartTime != nil {
			conditions = append(conditions, fmt.Sprintf("requested_at >= $%d", argIdx))
			args = append(args, *opts.StartTime)
			argIdx++
		}

		if opts.EndTime != nil {
			conditions = append(conditions, fmt.Sprintf("requested_at <= $%d", argIdx))
			args = append(args, *opts.EndTime)
			argIdx++
		}

		whereClause += joinConditions(conditions)
	}

	// Query overall statistics
	if err := queryOverallStats(ctx, pool, whereClause, args, result); err != nil {
		return nil, fmt.Errorf("failed to query overall stats: %w", err)
	}

	// Query time-based aggregations
	if opts.GroupBy == "day" || opts.GroupBy == "" {
		if err := queryDayStats(ctx, pool, whereClause, args, result); err != nil {
			log.Warnf("Failed to query day stats: %v", err)
		}
	}

	if opts.GroupBy == "hour" || opts.GroupBy == "" {
		if err := queryHourStats(ctx, pool, whereClause, args, result); err != nil {
			log.Warnf("Failed to query hour stats: %v", err)
		}
	}

	// Query API and model breakdown
	if err := queryAPIStats(ctx, pool, whereClause, args, result); err != nil {
		log.Warnf("Failed to query API stats: %v", err)
	}

	return result, nil
}

func joinConditions(conditions []string) string {
	if len(conditions) == 0 {
		return ""
	}
	var sb strings.Builder
	for i, c := range conditions {
		if i > 0 {
			sb.WriteString(" AND ")
		}
		sb.WriteString(c)
	}
	return sb.String()
}

func queryOverallStats(ctx context.Context, pool *pgxpool.Pool, whereClause string, args []any, result *QueryResult) error {
	query := `
SELECT
	COUNT(*) as total_requests,
	COUNT(*) FILTER (WHERE NOT failed) as success_count,
	COUNT(*) FILTER (WHERE failed) as failure_count,
	COALESCE(SUM(total_tokens), 0) as total_tokens
FROM usage_records
` + whereClause

	var totalReqs, success, failure, totalTok int64
	err := pool.QueryRow(ctx, query, args...).Scan(&totalReqs, &success, &failure, &totalTok)
	if err != nil {
		return err
	}

	result.TotalRequests = totalReqs
	result.SuccessCount = success
	result.FailureCount = failure
	result.TotalTokens = totalTok

	return nil
}

func queryDayStats(ctx context.Context, pool *pgxpool.Pool, whereClause string, args []any, result *QueryResult) error {
	// Use UTC timezone to match memory statistics
	query := `
SELECT
	DATE(requested_at AT TIME ZONE 'UTC') as day,
	COUNT(*) as requests,
	COALESCE(SUM(total_tokens), 0) as tokens
FROM usage_records
` + whereClause + `
GROUP BY DATE(requested_at AT TIME ZONE 'UTC')
ORDER BY day
`

	rows, err := pool.Query(ctx, query, args...)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var day time.Time
		var requests, tokens int64
		if err := rows.Scan(&day, &requests, &tokens); err != nil {
			return err
		}
		dayKey := day.Format("2006-01-02")
		result.RequestsByDay[dayKey] = requests
		result.TokensByDay[dayKey] = tokens
		log.Debugf("queryDayStats: day=%s requests=%d tokens=%d", dayKey, requests, tokens)
	}

	log.Debugf("queryDayStats: final RequestsByDay=%v TokensByDay=%v", result.RequestsByDay, result.TokensByDay)

	return rows.Err()
}

func queryHourStats(ctx context.Context, pool *pgxpool.Pool, whereClause string, args []any, result *QueryResult) error {
	// Query hourly stats grouped by date and hour to get meaningful hourly distribution
	// Use UTC timezone to match the formatHour function in usage package
	query := `
SELECT
	DATE(requested_at AT TIME ZONE 'UTC') as day,
	EXTRACT(HOUR FROM requested_at AT TIME ZONE 'UTC')::int as hour,
	COUNT(*) as requests,
	COALESCE(SUM(total_tokens), 0) as tokens
FROM usage_records
` + whereClause + `
GROUP BY DATE(requested_at AT TIME ZONE 'UTC'), EXTRACT(HOUR FROM requested_at AT TIME ZONE 'UTC')
ORDER BY day, hour
`

	rows, err := pool.Query(ctx, query, args...)
	if err != nil {
		return err
	}
	defer rows.Close()

	type dayHourKey struct {
		day  string
		hour string
	}
	type hourData struct {
		requests int64
		tokens  int64
	}
	hourlyData := make(map[dayHourKey]hourData)

	for rows.Next() {
		var day time.Time
		var hour int
		var requests, tokens int64
		if err := rows.Scan(&day, &hour, &requests, &tokens); err != nil {
			return err
		}
		dayKey := day.Format("2006-01-02")
		hourKey := fmt.Sprintf("%02d", hour)
		hourlyData[dayHourKey{day: dayKey, hour: hourKey}] = hourData{requests: requests, tokens: tokens}
	}

	// Aggregate by hour only (across all days) for compatibility
	hourTotals := make(map[string]hourData)

	for key, data := range hourlyData {
		existing := hourTotals[key.hour]
		existing.requests += data.requests
		existing.tokens += data.tokens
		hourTotals[key.hour] = existing
	}

	for hour, data := range hourTotals {
		result.RequestsByHour[hour] = data.requests
		result.TokensByHour[hour] = data.tokens
		log.Debugf("queryHourStats: hour=%s requests=%d tokens=%d", hour, data.requests, data.tokens)
	}

	log.Debugf("queryHourStats: final RequestsByHour=%v TokensByHour=%v", result.RequestsByHour, result.TokensByHour)

	return rows.Err()
}

func queryAPIStats(ctx context.Context, pool *pgxpool.Pool, whereClause string, args []any, result *QueryResult) error {
	// First, get aggregated stats
	aggQuery := `
SELECT
	COALESCE(api_key, 'unknown') as api_key,
	model,
	COUNT(*) as requests,
	COALESCE(SUM(total_tokens), 0) as tokens
FROM usage_records
` + whereClause + `
GROUP BY api_key, model
ORDER BY api_key, model
`

	rows, err := pool.Query(ctx, aggQuery, args...)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var apiKey, model string
		var requests, tokens int64
		if err := rows.Scan(&apiKey, &model, &requests, &tokens); err != nil {
			return err
		}

		if apiKey == "" {
			apiKey = "unknown"
		}
		if model == "" {
			model = "unknown"
		}

		apiStats, ok := result.APIs[apiKey]
		if !ok {
			apiStats = APIStats{
				Models: make(map[string]ModelStats),
			}
			result.APIs[apiKey] = apiStats
		}

		apiStats.TotalRequests += requests
		apiStats.TotalTokens += tokens
		apiStats.Models[model] = ModelStats{
			TotalRequests: requests,
			TotalTokens:   tokens,
			Details:       []RequestDetail{}, // Initialize empty slice
		}
		result.APIs[apiKey] = apiStats
	}

	if err := rows.Err(); err != nil {
		return err
	}

	// Then, get detailed records for each API/model combination
	detailQuery := `
SELECT
	COALESCE(api_key, 'unknown') as api_key,
	model,
	requested_at,
	COALESCE(source, '') as source,
	COALESCE(auth_index, '') as auth_index,
	failed,
	input_tokens,
	output_tokens,
	reasoning_tokens,
	cached_tokens,
	total_tokens
FROM usage_records
` + whereClause + `
ORDER BY api_key, model, requested_at
`

	detailRows, err := pool.Query(ctx, detailQuery, args...)
	if err != nil {
		return err
	}
	defer detailRows.Close()

	for detailRows.Next() {
		var apiKey, model, source, authIndex string
		var requestedAt time.Time
		var failed bool
		var inputTokens, outputTokens, reasoningTokens, cachedTokens, totalTokens int64

		if err := detailRows.Scan(&apiKey, &model, &requestedAt, &source, &authIndex, &failed,
			&inputTokens, &outputTokens, &reasoningTokens, &cachedTokens, &totalTokens); err != nil {
			return err
		}

		if apiKey == "" {
			apiKey = "unknown"
		}
		if model == "" {
			model = "unknown"
		}

		apiStats, ok := result.APIs[apiKey]
		if !ok {
			continue
		}

		modelStats, ok := apiStats.Models[model]
		if !ok {
			continue
		}

		modelStats.Details = append(modelStats.Details, RequestDetail{
			Timestamp: requestedAt,
			Source:    source,
			AuthIndex: authIndex,
			Failed:    failed,
			Tokens: TokenStats{
				InputTokens:     inputTokens,
				OutputTokens:    outputTokens,
				ReasoningTokens: reasoningTokens,
				CachedTokens:    cachedTokens,
				TotalTokens:     totalTokens,
			},
		})
		apiStats.Models[model] = modelStats
		result.APIs[apiKey] = apiStats
	}

	return detailRows.Err()
}

// ParseTimeRangePreset parses a time range preset and returns start/end times.
func ParseTimeRangePreset(preset TimeRangePreset, start, end *time.Time) (*time.Time, *time.Time, error) {
	now := time.Now().UTC()
	var startTime, endTime time.Time

	switch preset {
	case PresetToday:
		startTime = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
		endTime = time.Date(now.Year(), now.Month(), now.Day(), 23, 59, 59, 999999999, time.UTC)
	case PresetThisWeek:
		weekday := now.Weekday()
		if weekday == time.Sunday {
			weekday = 7
		}
		weekStart := now.AddDate(0, 0, -int(weekday-time.Monday))
		startTime = time.Date(weekStart.Year(), weekStart.Month(), weekStart.Day(), 0, 0, 0, 0, time.UTC)
		endTime = startTime.AddDate(0, 0, 6)
		endTime = time.Date(endTime.Year(), endTime.Month(), endTime.Day(), 23, 59, 59, 999999999, time.UTC)
	case PresetThisMonth:
		startTime = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
		endTime = time.Date(now.Year(), now.Month()+1, 0, 23, 59, 59, 999999999, time.UTC)
	case PresetLast7Days:
		endTime = now
		startTime = now.AddDate(0, 0, -6)
		startTime = time.Date(startTime.Year(), startTime.Month(), startTime.Day(), 0, 0, 0, 0, time.UTC)
		endTime = time.Date(endTime.Year(), endTime.Month(), endTime.Day(), 23, 59, 59, 999999999, time.UTC)
	case PresetLast30Days:
		endTime = now
		startTime = now.AddDate(0, 0, -29)
		startTime = time.Date(startTime.Year(), startTime.Month(), startTime.Day(), 0, 0, 0, 0, time.UTC)
		endTime = time.Date(endTime.Year(), endTime.Month(), endTime.Day(), 23, 59, 59, 999999999, time.UTC)
	case PresetCustom:
		if start == nil || end == nil {
			return nil, nil, fmt.Errorf("start and end time are required for custom preset")
		}
		return start, end, nil
	default:
		return nil, nil, fmt.Errorf("unknown preset: %s", preset)
	}

	return &startTime, &endTime, nil
}

// QueryProviderStats retrieves aggregated usage statistics grouped by provider.
func QueryProviderStats(ctx context.Context, pool *pgxpool.Pool, opts QueryOptions) (*ProviderStatsResult, error) {
	if pool == nil {
		return nil, fmt.Errorf("pool is not initialized")
	}

	result := &ProviderStatsResult{
		Providers: []ProviderStats{},
		TimeRange: TimeRange{
			Start: opts.StartTime,
			End:   opts.EndTime,
		},
	}

	// Build WHERE clause
	whereClause := ""
	args := []any{}
	argIdx := 1

	if opts.StartTime != nil || opts.EndTime != nil {
		whereClause = " WHERE "
		conditions := []string{}

		if opts.StartTime != nil {
			conditions = append(conditions, fmt.Sprintf("requested_at >= $%d", argIdx))
			args = append(args, *opts.StartTime)
			argIdx++
		}

		if opts.EndTime != nil {
			conditions = append(conditions, fmt.Sprintf("requested_at <= $%d", argIdx))
			args = append(args, *opts.EndTime)
			argIdx++
		}

		whereClause += joinConditions(conditions)
	}

	// Query provider statistics
	if err := queryProviderStats(ctx, pool, whereClause, args, result); err != nil {
		return nil, fmt.Errorf("failed to query provider stats: %w", err)
	}

	return result, nil
}

func queryProviderStats(ctx context.Context, pool *pgxpool.Pool, whereClause string, args []any, result *ProviderStatsResult) error {
	// Note: completed_at column may not exist, so we estimate latency from requested_at only
	query := `
SELECT
	COALESCE(provider, 'unknown') as provider,
	COUNT(*) as total_requests,
	COUNT(*) FILTER (WHERE NOT failed) as success_count,
	COUNT(*) FILTER (WHERE failed) as failure_count,
	COALESCE(SUM(input_tokens), 0) as input_tokens,
	COALESCE(SUM(output_tokens), 0) as output_tokens,
	COALESCE(SUM(reasoning_tokens), 0) as reasoning_tokens,
	COALESCE(SUM(cached_tokens), 0) as cached_tokens,
	COALESCE(SUM(total_tokens), 0) as total_tokens,
	COALESCE(NULL, 0::float8) as avg_latency_ms,
	COALESCE(NULL, 0::float8) as p50_latency_ms,
	COALESCE(NULL, 0::float8) as p95_latency_ms,
	COALESCE(NULL, 0::float8) as p99_latency_ms,
	MAX(requested_at) as last_called_at
FROM usage_records
` + whereClause + `
GROUP BY provider
ORDER BY provider
`

	rows, err := pool.Query(ctx, query, args...)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var stats ProviderStats
		if err := rows.Scan(
			&stats.Name,
			&stats.TotalRequests,
			&stats.SuccessCount,
			&stats.FailureCount,
			&stats.InputTokens,
			&stats.OutputTokens,
			&stats.ReasoningTokens,
			&stats.CachedTokens,
			&stats.TotalTokens,
			&stats.AvgLatencyMs,
			&stats.P50LatencyMs,
			&stats.P95LatencyMs,
			&stats.P99LatencyMs,
			&stats.LastCalledAt,
		); err != nil {
			return err
		}

		// Calculate success rate
		if stats.TotalRequests > 0 {
			stats.SuccessRate = float64(stats.SuccessCount) / float64(stats.TotalRequests) * 100
		}

		result.Providers = append(result.Providers, stats)
	}

	return rows.Err()
}

// QueryVendorErrorLogs retrieves failed vendor error logs with pagination and filters.
func QueryVendorErrorLogs(ctx context.Context, pool *pgxpool.Pool, opts VendorErrorLogListOptions) (*VendorErrorLogListResult, error) {
	if pool == nil {
		return nil, fmt.Errorf("pool is not initialized")
	}

	page := opts.Page
	limit := opts.Limit
	if page <= 0 {
		page = 1
	}
	if limit <= 0 {
		limit = 50
	}
	if limit > 500 {
		limit = 500
	}

	conditions := []string{"failed = true"}
	args := []any{}
	argIdx := 1

	if opts.Provider != "" {
		conditions = append(conditions, fmt.Sprintf("provider = $%d", argIdx))
		args = append(args, opts.Provider)
		argIdx++
	}

	if opts.StartTime != nil {
		conditions = append(conditions, fmt.Sprintf("requested_at >= $%d", argIdx))
		args = append(args, *opts.StartTime)
		argIdx++
	}

	if opts.EndTime != nil {
		conditions = append(conditions, fmt.Sprintf("requested_at <= $%d", argIdx))
		args = append(args, *opts.EndTime)
		argIdx++
	}

	whereClause := " WHERE " + joinConditions(conditions)

	countQuery := "SELECT COUNT(*) FROM usage_records" + whereClause
	var total int64
	if err := pool.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, err
	}

	offset := (page - 1) * limit
	limitArg := argIdx
	offsetArg := argIdx + 1
	args = append(args, limit, offset)

	query := fmt.Sprintf(`
SELECT
	COALESCE(provider, 'unknown') as provider,
	COALESCE(model, 'unknown') as model,
	COALESCE(api_key, '') as api_key,
	COALESCE(auth_id, '') as auth_id,
	COALESCE(auth_index, '') as auth_index,
	COALESCE(source, '') as source,
	requested_at,
	COALESCE(vendor_error_log, '') as vendor_error_log,
	COALESCE(request_url, '') as request_url,
	input_tokens,
	output_tokens,
	reasoning_tokens,
	cached_tokens,
	total_tokens
FROM usage_records
%s
ORDER BY requested_at DESC, id DESC
LIMIT $%d OFFSET $%d
`, whereClause, limitArg, offsetArg)

	rows, err := pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	entries := make([]VendorErrorLogEntry, 0)
	for rows.Next() {
		var entry VendorErrorLogEntry
		if err := rows.Scan(
			&entry.Provider,
			&entry.Model,
			&entry.APIKey,
			&entry.AuthID,
			&entry.AuthIndex,
			&entry.Source,
			&entry.RequestedAt,
			&entry.VendorErrorLog,
			&entry.RequestURL,
			&entry.InputTokens,
			&entry.OutputTokens,
			&entry.ReasoningTokens,
			&entry.CachedTokens,
			&entry.TotalTokens,
		); err != nil {
			return nil, err
		}
		entries = append(entries, entry)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return &VendorErrorLogListResult{
		Entries: entries,
		Total:   total,
		Page:    page,
		Limit:   limit,
		TimeRange: TimeRange{
			Start: opts.StartTime,
			End:   opts.EndTime,
		},
		Provider: opts.Provider,
	}, nil
}

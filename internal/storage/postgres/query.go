// Package postgres provides PostgreSQL storage backend for usage statistics.
package postgres

import (
	"context"
	"fmt"
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

// RequestDetail stores the timestamp and token usage for a single request.
type RequestDetail struct {
	Timestamp time.Time  `json:"timestamp"`
	Source    string     `json:"source"`
	AuthIndex string     `json:"auth_index"`
	Tokens    TokenStats `json:"tokens"`
	Failed    bool       `json:"failed"`
}

// TokenStats captures the token usage breakdown for a request.
type TokenStats struct {
	InputTokens     int64 `json:"input_tokens"`
	OutputTokens    int64 `json:"output_tokens"`
	ReasoningTokens int64 `json:"reasoning_tokens"`
	CachedTokens    int64 `json:"cached_tokens"`
	TotalTokens     int64 `json:"total_tokens"`
}

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
	args := []interface{}{}
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
	result := ""
	for i, c := range conditions {
		if i > 0 {
			result += " AND "
		}
		result += c
	}
	return result
}

func queryOverallStats(ctx context.Context, pool *pgxpool.Pool, whereClause string, args []interface{}, result *QueryResult) error {
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

func queryDayStats(ctx context.Context, pool *pgxpool.Pool, whereClause string, args []interface{}, result *QueryResult) error {
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

func queryHourStats(ctx context.Context, pool *pgxpool.Pool, whereClause string, args []interface{}, result *QueryResult) error {
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

func queryAPIStats(ctx context.Context, pool *pgxpool.Pool, whereClause string, args []interface{}, result *QueryResult) error {
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

package management

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
	"github.com/router-for-me/CLIProxyAPI/v6/internal/usage"
	"github.com/router-for-me/CLIProxyAPI/v6/internal/storage/postgres"
)

type usageExportPayload struct {
	Version    int                      `json:"version"`
	ExportedAt time.Time                `json:"exported_at"`
	Usage      usage.StatisticsSnapshot `json:"usage"`
}

type usageImportPayload struct {
	Version int                      `json:"version"`
	Usage   usage.StatisticsSnapshot `json:"usage"`
}

// GetUsageStatistics returns the usage statistics.
// When PostgreSQL storage is enabled, it queries from PostgreSQL by default.
// Use query parameter 'source' to explicitly select 'postgres' or 'memory'.
func (h *Handler) GetUsageStatistics(c *gin.Context) {
	source := c.Query("source")

	// If source is not specified, use PostgreSQL if enabled, otherwise memory
	if source == "" {
		log.Debugf("GetUsageStatistics: source not specified, checking postgres plugin status")
		if h != nil && h.postgresPlugin != nil && h.postgresPlugin.IsActive() {
			source = "postgres"
			log.Debugf("GetUsageStatistics: PostgreSQL is active, will query from postgres")
		} else {
			source = "memory"
			log.Debugf("GetUsageStatistics: PostgreSQL not active, will query from memory")
		}
	} else {
		log.Debugf("GetUsageStatistics: source explicitly set to '%s'", source)
	}

	// Handle PostgreSQL source
	if source == "postgres" {
		if h == nil || h.postgresPlugin == nil || !h.postgresPlugin.IsActive() {
			c.JSON(http.StatusBadRequest, gin.H{"error": "PostgreSQL storage is not enabled"})
			return
		}

		// Type assertion to get the plugin
		plugin, ok := h.postgresPlugin.(*postgres.Plugin)
		if !ok || plugin.Pool() == nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "PostgreSQL plugin unavailable"})
			return
		}

		// Parse query options
		opts := postgres.QueryOptions{
			GroupBy: c.Query("group_by"),
		}

		if startStr := c.Query("start"); startStr != "" {
			if t, err := time.Parse(time.RFC3339, startStr); err == nil {
				opts.StartTime = &t
			} else if t, err := time.Parse("2006-01-02", startStr); err == nil {
				opts.StartTime = &t
			}
		}

		if endStr := c.Query("end"); endStr != "" {
			if t, err := time.Parse(time.RFC3339, endStr); err == nil {
				opts.EndTime = &t
			} else if t, err := time.Parse("2006-01-02", endStr); err == nil {
				// End of day
				t = t.Add(24*time.Hour - time.Second)
				opts.EndTime = &t
			}
		}

		// Query PostgreSQL
		log.Debugf("GetUsageStatistics: querying postgres with opts=%+v", opts)
		result, err := postgres.QueryStats(c.Request.Context(), plugin.Pool().Pool(), opts)
		log.Debugf("GetUsageStatistics: postgres query result: total=%d tokens=%d days=%d hours=%d",
			result.TotalRequests, result.TotalTokens, len(result.RequestsByDay), len(result.RequestsByHour))
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		// Convert to compatible format
		snapshot := usage.StatisticsSnapshot{
			TotalRequests:   result.TotalRequests,
			SuccessCount:    result.SuccessCount,
			FailureCount:    result.FailureCount,
			TotalTokens:     result.TotalTokens,
			RequestsByDay:   result.RequestsByDay,
			TokensByDay:     result.TokensByDay,
			RequestsByHour:  result.RequestsByHour,
			TokensByHour:    result.TokensByHour,
		}

		log.Debugf("GetUsageStatistics: snapshot RequestsByDay=%v RequestsByHour=%v", snapshot.RequestsByDay, snapshot.RequestsByHour)

		// Convert APIs to match the expected format
		snapshot.APIs = make(map[string]usage.APISnapshot)
		for key, apiStat := range result.APIs {
			apiSnap := usage.APISnapshot{
				TotalRequests: apiStat.TotalRequests,
				TotalTokens:   apiStat.TotalTokens,
				Models:        make(map[string]usage.ModelSnapshot),
			}
			for modelKey, modelStat := range apiStat.Models {
				// Convert details from postgres format to usage format
				details := make([]usage.RequestDetail, len(modelStat.Details))
				for i, d := range modelStat.Details {
					details[i] = usage.RequestDetail{
						Timestamp: d.Timestamp,
						Source:    d.Source,
						AuthIndex: d.AuthIndex,
						Failed:    d.Failed,
						Tokens: usage.TokenStats{
							InputTokens:     d.Tokens.InputTokens,
							OutputTokens:    d.Tokens.OutputTokens,
							ReasoningTokens: d.Tokens.ReasoningTokens,
							CachedTokens:    d.Tokens.CachedTokens,
							TotalTokens:     d.Tokens.TotalTokens,
						},
					}
				}
				apiSnap.Models[modelKey] = usage.ModelSnapshot{
					TotalRequests: modelStat.TotalRequests,
					TotalTokens:   modelStat.TotalTokens,
					Details:       details,
				}
			}
			snapshot.APIs[key] = apiSnap
		}

		c.JSON(http.StatusOK, gin.H{
			"usage":           snapshot,
			"failed_requests": snapshot.FailureCount,
			"source":          "postgres",
		})
		return
	}

	// Default: in-memory source
	var snapshot usage.StatisticsSnapshot
	if h != nil && h.usageStats != nil {
		snapshot = h.usageStats.Snapshot()
	}
	c.JSON(http.StatusOK, gin.H{
		"usage":           snapshot,
		"failed_requests": snapshot.FailureCount,
		"source":          "memory",
	})
}

// ExportUsageStatistics returns a complete usage snapshot for backup/migration.
func (h *Handler) ExportUsageStatistics(c *gin.Context) {
	var snapshot usage.StatisticsSnapshot
	if h != nil && h.usageStats != nil {
		snapshot = h.usageStats.Snapshot()
	}
	c.JSON(http.StatusOK, usageExportPayload{
		Version:    1,
		ExportedAt: time.Now().UTC(),
		Usage:      snapshot,
	})
}

// ImportUsageStatistics merges a previously exported usage snapshot into memory.
func (h *Handler) ImportUsageStatistics(c *gin.Context) {
	if h == nil || h.usageStats == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "usage statistics unavailable"})
		return
	}

	data, err := c.GetRawData()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to read request body"})
		return
	}

	var payload usageImportPayload
	if err := json.Unmarshal(data, &payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid json"})
		return
	}
	if payload.Version != 0 && payload.Version != 1 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "unsupported version"})
		return
	}

	result := h.usageStats.MergeSnapshot(payload.Usage)
	snapshot := h.usageStats.Snapshot()
	c.JSON(http.StatusOK, gin.H{
		"added":           result.Added,
		"skipped":         result.Skipped,
		"total_requests":  snapshot.TotalRequests,
		"failed_requests": snapshot.FailureCount,
	})
}

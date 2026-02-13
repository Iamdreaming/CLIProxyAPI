// Package management provides the management API handlers.
package management

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/router-for-me/CLIProxyAPI/v6/internal/storage/postgres"
	log "github.com/sirupsen/logrus"
)

type postgresPoolProvider interface {
	IsActive() bool
	Pool() *postgres.Pool
}

var queryVendorErrorLogs = postgres.QueryVendorErrorLogs


// GetProviderStats returns aggregated statistics grouped by provider.
// Query parameters:
//   - preset: Time range preset (today, this_week, this_month, last_7_days, last_30_days, custom)
//   - start: Start time (RFC3339 or YYYY-MM-DD format) - required for custom preset
//   - end: End time (RFC3339 or YYYY-MM-DD format) - required for custom preset
func (h *Handler) GetProviderStats(c *gin.Context) {
	// Determine data source
	source := "memory"
	var result *postgres.ProviderStatsResult

	// Check if PostgreSQL is available
	pgActive := h != nil && h.postgresPlugin != nil && h.postgresPlugin.IsActive()

	if pgActive {
		plugin, ok := h.postgresPlugin.(postgresPoolProvider)
		if !ok || plugin.Pool() == nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "PostgreSQL plugin unavailable"})
			return
		}

		// Parse time range
		startTime, endTime, err := parseTimeRangeParams(c)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		opts := postgres.QueryOptions{
			StartTime: startTime,
			EndTime:   endTime,
		}

		log.Debugf("GetProviderStats: querying postgres with opts=%+v", opts)
		pgResult, err := postgres.QueryProviderStats(c.Request.Context(), plugin.Pool().Pool(), opts)
		if err != nil {
			log.Errorf("GetProviderStats: failed to query postgres: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		// Build response directly from provider stats result
		result = pgResult
		source = "postgres"
	} else {
		// Memory source not supported for provider-level stats yet
		// Return empty result with proper structure
		result = &postgres.ProviderStatsResult{
			Providers: []postgres.ProviderStats{},
			TimeRange: postgres.TimeRange{},
		}
		source = "memory"
	}

	c.JSON(http.StatusOK, gin.H{
		"providers":  result.Providers,
		"time_range": result.TimeRange,
		"source":     source,
	})
}

// GetVendorErrorLogs returns failed vendor error logs with pagination and filters.
// Query parameters:
//   - provider: Filter by vendor/provider name
//   - preset: Time range preset (today, this_week, this_month, last_7_days, last_30_days, custom)
//   - start: Start time (RFC3339 or YYYY-MM-DD format) - required for custom preset
//   - end: End time (RFC3339 or YYYY-MM-DD format) - required for custom preset
//   - page: Page number (1-based)
//   - limit: Page size (max 500)
func (h *Handler) GetVendorErrorLogs(c *gin.Context) {
	if h == nil || h.postgresPlugin == nil || !h.postgresPlugin.IsActive() {
		c.JSON(http.StatusBadRequest, gin.H{"error": "PostgreSQL storage is not enabled"})
		return
	}

	plugin, ok := h.postgresPlugin.(postgresPoolProvider)
	if !ok || plugin.Pool() == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "PostgreSQL plugin unavailable"})
		return
	}

	startTime, endTime, err := parseTimeRangeParams(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	page, err := parsePositiveInt(c.Query("page"), 1)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	limit, err := parsePositiveInt(c.Query("limit"), 50)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	provider := strings.TrimSpace(c.Query("provider"))

	opts := postgres.VendorErrorLogListOptions{
		StartTime: startTime,
		EndTime:   endTime,
		Provider:  provider,
		Page:      page,
		Limit:     limit,
	}

	result, err := queryVendorErrorLogs(c.Request.Context(), plugin.Pool().Pool(), opts)
	if err != nil {
		log.Errorf("GetVendorErrorLogs: failed to query postgres: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"entries":    result.Entries,
		"total":      result.Total,
		"page":       result.Page,
		"limit":      result.Limit,
		"time_range": result.TimeRange,
		"provider":   result.Provider,
		"source":     "postgres",
	})
}

func parsePositiveInt(raw string, defaultValue int) (int, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return defaultValue, nil
	}
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed <= 0 {
		return 0, fmt.Errorf("invalid value: %s", raw)
	}
	return parsed, nil
}

// parseTimeRangeParams parses time range from query parameters.
func parseTimeRangeParams(c *gin.Context) (*time.Time, *time.Time, error) {
	preset := postgres.TimeRangePreset(c.Query("preset"))

	var startTime, endTime *time.Time
	var err error

	if preset != "" {
		// Handle presets
		startTime, endTime, err = postgres.ParseTimeRangePreset(preset, nil, nil)
		if err != nil {
			return nil, nil, err
		}
	} else {
		// Parse explicit start/end times
		if startStr := c.Query("start"); startStr != "" {
			if t, err := time.Parse(time.RFC3339, startStr); err == nil {
				startTime = &t
			} else if t, err := time.Parse("2006-01-02", startStr); err == nil {
				startTime = &t
			}
		}

		if endStr := c.Query("end"); endStr != "" {
			if t, err := time.Parse(time.RFC3339, endStr); err == nil {
				endTime = &t
			} else if t, err := time.Parse("2006-01-02", endStr); err == nil {
				// End of day
				t = t.Add(24*time.Hour - time.Second)
				endTime = &t
			}
		}
	}

	return startTime, endTime, nil
}
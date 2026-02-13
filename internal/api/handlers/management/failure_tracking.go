package management

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/router-for-me/CLIProxyAPI/v6/internal/failure"
)

// FailureTrackerSetter is an interface for setting the failure tracker.
type FailureTrackerSetter interface {
	SetFailureTracker(tracker failure.FailureTracker)
}

// SetFailureTracker sets the failure tracker for the handler.
func (h *Handler) SetFailureTracker(tracker failure.FailureTracker) {
	// This will be stored in a new field that we need to add
	h.setFailureTracker(tracker)
}

// setFailureTracker is the internal implementation.
func (h *Handler) setFailureTracker(tracker failure.FailureTracker) {
	// We'll use the integration helper to wrap the tracker
	if tracker != nil {
		h.failureIntegration = failure.NewIntegration(tracker)
	} else {
		h.failureIntegration = nil
	}
}

// GetDisabledModels returns a list of all currently disabled models.
func (h *Handler) GetDisabledModels(c *gin.Context) {
	if h.failureIntegration == nil {
		c.JSON(http.StatusOK, gin.H{
			"models":     []interface{}{},
			"message":   "failure tracking not enabled",
		})
		return
	}

	disabled := h.failureIntegration.GetAllDisabledModels()
	models := make([]gin.H, 0, len(disabled))

	for _, dm := range disabled {
		models = append(models, gin.H{
			"vendor":           dm.Vendor,
			"model":            dm.Model,
			"failureCount":     dm.FailureCount,
			"disabledAt":       dm.DisabledAt,
			"disabledUntil":    dm.DisabledUntil,
			"remainingSeconds": int(dm.RemainingTime.Seconds()),
			"failureThreshold": dm.FailureThreshold,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"models": models,
		"count":  len(models),
	})
}

// GetModelStatus returns the status (enabled/disabled) for a specific model.
func (h *Handler) GetModelStatus(c *gin.Context) {
	modelID := c.Param("modelId")
	if modelID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "modelId is required"})
		return
	}

	// Parse vendor:model from the ID
	parts := strings.SplitN(modelID, ":", 2)
	if len(parts) != 2 {
		// Try to handle it as just a model name with unknown vendor
		c.JSON(http.StatusBadRequest, gin.H{"error": "modelId must be in format 'vendor:model'"})
		return
	}
	vendor := parts[0]
	model := parts[1]

	if h.failureIntegration == nil {
		c.JSON(http.StatusOK, gin.H{
			"vendor":        vendor,
			"model":         model,
			"disabled":      false,
			"failureCount":  0,
			"message":       "failure tracking not enabled",
		})
		return
	}

	disabled, _ := h.failureIntegration.IsModelDisabled(vendor, model)
	failureCount, _ := h.failureIntegration.GetFailureCount(vendor, model)

	c.JSON(http.StatusOK, gin.H{
		"vendor":        vendor,
		"model":         model,
		"disabled":      disabled,
		"failureCount":  failureCount,
	})
}

// EnableModel manually re-enables a previously disabled model.
func (h *Handler) EnableModel(c *gin.Context) {
	modelID := c.Param("modelId")
	if modelID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "modelId is required"})
		return
	}

	// Parse vendor:model from the ID
	parts := strings.SplitN(modelID, ":", 2)
	if len(parts) != 2 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "modelId must be in format 'vendor:model'"})
		return
	}
	vendor := parts[0]
	model := parts[1]

	if h.failureIntegration == nil {
		c.JSON(http.StatusOK, gin.H{
			"status":  "noop",
			"message": "failure tracking not enabled",
		})
		return
	}

	err := h.failureIntegration.EnableModel(vendor, model)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"vendor":  vendor,
		"model":   model,
		"message": "model re-enabled successfully",
	})
}

// EnableAllModels re-enables all currently disabled models.
func (h *Handler) EnableAllModels(c *gin.Context) {
	if h.failureIntegration == nil {
		c.JSON(http.StatusOK, gin.H{
			"status":  "noop",
			"message": "failure tracking not enabled",
		})
		return
	}

	disabled := h.failureIntegration.GetAllDisabledModels()
	count := 0

	for _, dm := range disabled {
		if err := h.failureIntegration.EnableModel(dm.Vendor, dm.Model); err != nil {
			continue
		}
		count++
	}

	c.JSON(http.StatusOK, gin.H{
		"status":      "success",
		"enabledCount": count,
		"totalModels": len(disabled),
	})
}

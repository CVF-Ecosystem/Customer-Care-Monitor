package handlers

import (
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/engine"
	"github.com/gin-gonic/gin"
	"net/http"
)

// Read-only process capability, behind tenant read permission. No endpoint,
// credential, provider configuration, or customer data is returned.
func GetInterventionCapability(c *gin.Context) {
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, gin.H{
		"scope":                 "synthetic_only",
		"generation_configured": InterventionRunner.SyntheticConfigured(),
		"rerun_configured":      engine.WaitDataExecutor.SyntheticConfigured(),
	})
}

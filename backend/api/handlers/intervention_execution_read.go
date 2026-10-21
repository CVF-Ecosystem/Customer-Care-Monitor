package handlers

import (
	"errors"
	"net/http"

	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/api/middleware"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/engine"
	"github.com/gin-gonic/gin"
)

func executionReadError(c *gin.Context, err error, unavailable string) {
	switch {
	case errors.Is(err, engine.ErrExecutionReadInvalid):
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request"})
	case errors.Is(err, engine.ErrExecutionReadForbidden):
		c.JSON(http.StatusForbidden, gin.H{"error": "permission_denied"})
	case errors.Is(err, engine.ErrExecutionReadMissing):
		c.JSON(http.StatusNotFound, gin.H{"error": "not_found"})
	default:
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": unavailable})
	}
}

func GetInterventionExecutionReadiness(c *gin.Context) {
	result, err := engine.ReadExecutionReadiness(c.Request.Context(), middleware.GetTenantID(c), middleware.GetUserID(c), c.Param("caseId"), c.Param("requestId"), engine.WaitDataExecutor)
	if err != nil {
		executionReadError(c, err, "readiness_unavailable")
		return
	}
	c.JSON(http.StatusOK, result)
}

func GetInterventionExecutionStatus(c *gin.Context) {
	result, err := engine.ReadExecutionStatus(c.Request.Context(), middleware.GetTenantID(c), middleware.GetUserID(c), c.Param("caseId"), c.Param("requestId"))
	if err != nil {
		executionReadError(c, err, "status_unavailable")
		return
	}
	c.JSON(http.StatusOK, result)
}

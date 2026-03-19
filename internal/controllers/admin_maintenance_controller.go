package controllers

import (
	"context"

	httpx "awesomeproject/internal/platform/httpx"

	"github.com/gin-gonic/gin"
)

type AdminMaintenanceService interface {
	Reindex(ctx context.Context) error
}

type AdminMaintenanceController struct {
	service AdminMaintenanceService
}

func NewAdminMaintenanceController(service AdminMaintenanceService) *AdminMaintenanceController {
	return &AdminMaintenanceController{service: service}
}

func (ctl *AdminMaintenanceController) ReindexSearch(c *gin.Context) {
	if err := ctl.service.Reindex(c.Request.Context()); err != nil {
		httpx.FailError(c, err)
		return
	}
	httpx.Success(c, gin.H{"status": "reindexed"})
}

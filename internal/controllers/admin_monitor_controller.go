package controllers

import (
	httpx "awesomeproject/internal/platform/httpx"
	"awesomeproject/pkg/contracts"

	"github.com/gin-gonic/gin"
)

type AdminMonitorService interface {
	Overview() *contracts.MonitorOverview
	Timeseries() *contracts.MonitorTimeseries
}

type AdminMonitorController struct {
	service AdminMonitorService
}

func NewAdminMonitorController(service AdminMonitorService) *AdminMonitorController {
	return &AdminMonitorController{service: service}
}

func (ctl *AdminMonitorController) Overview(c *gin.Context) {
	httpx.Success(c, ctl.service.Overview())
}

func (ctl *AdminMonitorController) Timeseries(c *gin.Context) {
	httpx.Success(c, ctl.service.Timeseries())
}

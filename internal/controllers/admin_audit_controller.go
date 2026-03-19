package controllers

import (
	httpx "awesomeproject/internal/platform/httpx"
	"awesomeproject/pkg/contracts"

	"github.com/gin-gonic/gin"
)

type AdminAuditService interface {
	ListAuditLogs(query contracts.ListAIAuditLogsQuery) ([]contracts.AIAuditLogDTO, error)
	GetAuditLog(id uint64) (*contracts.AIAuditLogDetailDTO, error)
}

type AdminAuditController struct {
	service AdminAuditService
}

func NewAdminAuditController(service AdminAuditService) *AdminAuditController {
	return &AdminAuditController{service: service}
}

func (ctl *AdminAuditController) List(c *gin.Context) {
	var query contracts.ListAIAuditLogsQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		httpx.Fail(c, 400, err.Error())
		return
	}
	rows, err := ctl.service.ListAuditLogs(query)
	if err != nil {
		httpx.FailError(c, err)
		return
	}
	httpx.Success(c, rows)
}

func (ctl *AdminAuditController) Detail(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		httpx.Fail(c, 400, err.Error())
		return
	}
	row, err := ctl.service.GetAuditLog(id)
	if err != nil {
		httpx.FailError(c, err)
		return
	}
	httpx.Success(c, row)
}

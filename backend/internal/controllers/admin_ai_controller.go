package controllers

import (
	httpx "github.com/LeeJc02/WeHi/backend/internal/platform/httpx"
	"github.com/LeeJc02/WeHi/backend/pkg/contracts"

	"github.com/gin-gonic/gin"
)

type AdminAIConfigService interface {
	Load() (*contracts.AIConfig, error)
	Save(cfg contracts.AIConfig) error
}

type AdminAIRetryService interface {
	ListRetryJobs(query contracts.ListAIRetryJobsQuery) ([]contracts.AIRetryJobDTO, error)
	GetRetryJob(id uint64) (*contracts.AIRetryJobDetailDTO, error)
	RetryJobNow(id uint64) error
	RetryJobs(ids []uint64) error
	CleanupRetryJobs(statuses []string) error
}

type AdminAIController struct {
	configService AdminAIConfigService
	retryService  AdminAIRetryService
}

func NewAdminAIController(configService AdminAIConfigService, retryService AdminAIRetryService) *AdminAIController {
	return &AdminAIController{configService: configService, retryService: retryService}
}

func (ctl *AdminAIController) GetConfig(c *gin.Context) {
	cfg, err := ctl.configService.Load()
	if err != nil {
		httpx.FailError(c, err)
		return
	}
	httpx.Success(c, cfg)
}

func (ctl *AdminAIController) UpdateConfig(c *gin.Context) {
	var req contracts.AIConfig
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Fail(c, 400, err.Error())
		return
	}
	if err := ctl.configService.Save(req); err != nil {
		httpx.FailError(c, err)
		return
	}
	httpx.Success(c, req)
}

func (ctl *AdminAIController) ListRetryJobs(c *gin.Context) {
	var query contracts.ListAIRetryJobsQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		httpx.Fail(c, 400, err.Error())
		return
	}
	rows, err := ctl.retryService.ListRetryJobs(query)
	if err != nil {
		httpx.FailError(c, err)
		return
	}
	httpx.Success(c, rows)
}

func (ctl *AdminAIController) RetryJobDetail(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		httpx.Fail(c, 400, err.Error())
		return
	}
	row, err := ctl.retryService.GetRetryJob(id)
	if err != nil {
		httpx.FailError(c, err)
		return
	}
	httpx.Success(c, row)
}

func (ctl *AdminAIController) RetryJobNow(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		httpx.Fail(c, 400, err.Error())
		return
	}
	if err := ctl.retryService.RetryJobNow(id); err != nil {
		httpx.FailError(c, err)
		return
	}
	httpx.Success(c, gin.H{"status": "queued"})
}

func (ctl *AdminAIController) RetryJobs(c *gin.Context) {
	var req contracts.RetryAIRetryJobsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Fail(c, 400, err.Error())
		return
	}
	if err := ctl.retryService.RetryJobs(req.IDs); err != nil {
		httpx.FailError(c, err)
		return
	}
	httpx.Success(c, gin.H{"status": "queued"})
}

func (ctl *AdminAIController) CleanupRetryJobs(c *gin.Context) {
	var req contracts.CleanupAIRetryJobsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Fail(c, 400, err.Error())
		return
	}
	if err := ctl.retryService.CleanupRetryJobs(req.Statuses); err != nil {
		httpx.FailError(c, err)
		return
	}
	httpx.Success(c, gin.H{"status": "cleaned"})
}

package controllers

import (
	httpx "github.com/LeeJc02/WeHi/backend/internal/platform/httpx"
	"github.com/LeeJc02/WeHi/backend/pkg/contracts"

	"github.com/gin-gonic/gin"
)

type SyncService interface {
	CurrentCursor(userID uint64) (*contracts.SyncCursorResponse, error)
	ListEvents(userID, cursor uint64, limit int) (*contracts.SyncEventsResponse, error)
}

type SyncController struct {
	service SyncService
}

func NewSyncController(service SyncService) *SyncController {
	return &SyncController{service: service}
}

func (ctl *SyncController) Cursor(c *gin.Context) {
	user, _, ok := requireCurrentUser(c)
	if !ok {
		return
	}
	result, err := ctl.service.CurrentCursor(user.ID)
	if err != nil {
		httpx.FailError(c, err)
		return
	}
	httpx.Success(c, result)
}

func (ctl *SyncController) Events(c *gin.Context) {
	user, _, ok := requireCurrentUser(c)
	if !ok {
		return
	}
	var query contracts.SyncEventsQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		httpx.Fail(c, 400, err.Error())
		return
	}
	result, err := ctl.service.ListEvents(user.ID, query.Cursor, query.Limit)
	if err != nil {
		httpx.FailError(c, err)
		return
	}
	httpx.Success(c, result)
}

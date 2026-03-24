package controllers

import (
	"context"

	httpx "github.com/LeeJc02/WeHi/backend/internal/platform/httpx"
	"github.com/LeeJc02/WeHi/backend/pkg/contracts"

	"github.com/gin-gonic/gin"
)

type SearchService interface {
	Search(ctx context.Context, userID uint64, query, scope, cursor string, limit int, conversationID uint64) (*contracts.SearchResponse, error)
}

type SearchController struct {
	service SearchService
}

func NewSearchController(service SearchService) *SearchController {
	return &SearchController{service: service}
}

func (ctl *SearchController) Search(c *gin.Context) {
	user, _, ok := requireCurrentUser(c)
	if !ok {
		return
	}
	var query contracts.SearchQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		httpx.Fail(c, 400, err.Error())
		return
	}
	if query.Limit == 0 {
		query.Limit = 10
	}
	if query.Scope == "" {
		query.Scope = "all"
	}
	result, err := ctl.service.Search(c.Request.Context(), user.ID, query.Q, query.Scope, query.Cursor, query.Limit, query.ConversationID)
	if err != nil {
		httpx.FailError(c, err)
		return
	}
	httpx.Success(c, result)
}

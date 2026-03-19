package controllers

import (
	httpx "awesomeproject/internal/platform/httpx"
	"awesomeproject/pkg/contracts"

	"github.com/gin-gonic/gin"
)

type AdminDiagnosticsService interface {
	MessageJourney(messageID uint64) (*contracts.MessageJourney, error)
	ResolveMessageByClientMsgID(clientMsgID string, senderID, conversationID uint64) (*contracts.MessageLookupResult, error)
	ConversationConsistency(conversationID uint64) (*contracts.ConversationConsistency, error)
	ConversationEvents(conversationID uint64, limit int) ([]contracts.SyncEventDTO, error)
}

type AdminDiagnosticsController struct {
	service AdminDiagnosticsService
}

func NewAdminDiagnosticsController(service AdminDiagnosticsService) *AdminDiagnosticsController {
	return &AdminDiagnosticsController{service: service}
}

func (ctl *AdminDiagnosticsController) MessageJourney(c *gin.Context) {
	messageID, err := parseUintParam(c, "messageId")
	if err != nil {
		httpx.Fail(c, 400, err.Error())
		return
	}
	payload, err := ctl.service.MessageJourney(messageID)
	if err != nil {
		httpx.FailError(c, err)
		return
	}
	httpx.Success(c, payload)
}

func (ctl *AdminDiagnosticsController) ResolveMessage(c *gin.Context) {
	var query struct {
		ClientMsgID    string `form:"client_msg_id"`
		SenderID       uint64 `form:"sender_id"`
		ConversationID uint64 `form:"conversation_id"`
	}
	if err := c.ShouldBindQuery(&query); err != nil {
		httpx.Fail(c, 400, err.Error())
		return
	}
	payload, err := ctl.service.ResolveMessageByClientMsgID(query.ClientMsgID, query.SenderID, query.ConversationID)
	if err != nil {
		httpx.FailError(c, err)
		return
	}
	httpx.Success(c, payload)
}

func (ctl *AdminDiagnosticsController) ConversationConsistency(c *gin.Context) {
	conversationID, err := parseUintParam(c, "id")
	if err != nil {
		httpx.Fail(c, 400, err.Error())
		return
	}
	payload, err := ctl.service.ConversationConsistency(conversationID)
	if err != nil {
		httpx.FailError(c, err)
		return
	}
	httpx.Success(c, payload)
}

func (ctl *AdminDiagnosticsController) ConversationEvents(c *gin.Context) {
	conversationID, err := parseUintParam(c, "id")
	if err != nil {
		httpx.Fail(c, 400, err.Error())
		return
	}
	var query struct {
		Limit int `form:"limit"`
	}
	if err := c.ShouldBindQuery(&query); err != nil {
		httpx.Fail(c, 400, err.Error())
		return
	}
	payload, err := ctl.service.ConversationEvents(conversationID, query.Limit)
	if err != nil {
		httpx.FailError(c, err)
		return
	}
	httpx.Success(c, payload)
}

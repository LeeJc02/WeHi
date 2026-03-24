package controllers

import (
	"strings"

	httpx "github.com/LeeJc02/WeHi/backend/internal/platform/httpx"
	"github.com/LeeJc02/WeHi/backend/pkg/contracts"

	"github.com/gin-gonic/gin"
)

type MessageService interface {
	ListMessages(userID, conversationID uint64, cursor string, limit int) ([]contracts.MessageDTO, error)
	SendMessage(userID, conversationID uint64, messageType, content, clientMsgID string, replyToMessageID *uint64, attachment *contracts.AttachmentDTO) (*contracts.MessageDTO, error)
	MarkRead(userID, conversationID uint64, seq uint64) error
	UpdateTyping(userID, conversationID uint64, isTyping bool) error
	RecallMessage(userID, messageID uint64) error
}

type MessageController struct {
	service MessageService
}

func NewMessageController(service MessageService) *MessageController {
	return &MessageController{service: service}
}

func (ctl *MessageController) ListMessages(c *gin.Context) {
	user, _, ok := requireCurrentUser(c)
	if !ok {
		return
	}
	conversationID, err := parseUintParam(c, "id")
	if err != nil {
		httpx.Fail(c, 400, err.Error())
		return
	}
	var query contracts.ListMessagesQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		httpx.Fail(c, 400, err.Error())
		return
	}
	if query.Limit == 0 {
		query.Limit = 20
	}
	messages, err := ctl.service.ListMessages(user.ID, conversationID, query.Cursor, query.Limit)
	if err != nil {
		httpx.FailError(c, err)
		return
	}
	httpx.Success(c, messages)
}

func (ctl *MessageController) SendMessage(c *gin.Context) {
	user, _, ok := requireCurrentUser(c)
	if !ok {
		return
	}
	conversationID, err := parseUintParam(c, "id")
	if err != nil {
		httpx.Fail(c, 400, err.Error())
		return
	}
	var req contracts.SendMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Fail(c, 400, err.Error())
		return
	}
	message, err := ctl.service.SendMessage(user.ID, conversationID, req.MessageType, req.Content, req.ClientMsgID, req.ReplyToMessageID, req.Attachment)
	if err != nil {
		httpx.FailError(c, err)
		return
	}
	httpx.Success(c, message)
}

func (ctl *MessageController) MarkRead(c *gin.Context) {
	user, _, ok := requireCurrentUser(c)
	if !ok {
		return
	}
	conversationID, err := parseUintParam(c, "id")
	if err != nil {
		httpx.Fail(c, 400, err.Error())
		return
	}
	var req contracts.MarkReadRequest
	if err := c.ShouldBindJSON(&req); err != nil && !strings.Contains(err.Error(), "EOF") {
		httpx.Fail(c, 400, err.Error())
		return
	}
	if err := ctl.service.MarkRead(user.ID, conversationID, req.Seq); err != nil {
		httpx.FailError(c, err)
		return
	}
	httpx.Success(c, gin.H{"status": "read"})
}

func (ctl *MessageController) Recall(c *gin.Context) {
	user, _, ok := requireCurrentUser(c)
	if !ok {
		return
	}
	messageID, err := parseUintParam(c, "id")
	if err != nil {
		httpx.Fail(c, 400, err.Error())
		return
	}
	if err := ctl.service.RecallMessage(user.ID, messageID); err != nil {
		httpx.FailError(c, err)
		return
	}
	httpx.Success(c, gin.H{"status": "recalled"})
}

func (ctl *MessageController) UpdateTyping(c *gin.Context) {
	user, _, ok := requireCurrentUser(c)
	if !ok {
		return
	}
	conversationID, err := parseUintParam(c, "id")
	if err != nil {
		httpx.Fail(c, 400, err.Error())
		return
	}
	var req contracts.TypingStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Fail(c, 400, err.Error())
		return
	}
	if err := ctl.service.UpdateTyping(user.ID, conversationID, req.IsTyping); err != nil {
		httpx.FailError(c, err)
		return
	}
	httpx.Success(c, gin.H{"status": "updated"})
}

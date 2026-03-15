package controllers

import (
	"net/http"
	"strconv"

	"awesomeproject/backend/internal/middleware"
	"awesomeproject/backend/internal/services"

	"github.com/gin-gonic/gin"
)

type ConversationController struct {
	conversations *services.ConversationService
}

func NewConversationController(conversations *services.ConversationService) *ConversationController {
	return &ConversationController{conversations: conversations}
}

func (ctl *ConversationController) List(c *gin.Context) {
	user, ok := middleware.CurrentUser(c)
	if !ok {
		fail(c, http.StatusUnauthorized, "unauthorized")
		return
	}
	conversations, err := ctl.conversations.ListConversations(user.ID)
	if err != nil {
		fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	success(c, conversations)
}

func (ctl *ConversationController) CreateDirect(c *gin.Context) {
	user, ok := middleware.CurrentUser(c)
	if !ok {
		fail(c, http.StatusUnauthorized, "unauthorized")
		return
	}
	var req struct {
		TargetUserID uint64 `json:"target_user_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, http.StatusBadRequest, err.Error())
		return
	}
	conversation, err := ctl.conversations.EnsureDirectConversation(user.ID, req.TargetUserID)
	if err != nil {
		fail(c, http.StatusBadRequest, err.Error())
		return
	}
	success(c, conversation)
}

func (ctl *ConversationController) CreateGroup(c *gin.Context) {
	user, ok := middleware.CurrentUser(c)
	if !ok {
		fail(c, http.StatusUnauthorized, "unauthorized")
		return
	}
	var req struct {
		Name      string   `json:"name" binding:"required"`
		MemberIDs []uint64 `json:"member_ids" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, http.StatusBadRequest, err.Error())
		return
	}
	conversation, err := ctl.conversations.CreateGroupConversation(user.ID, req.Name, req.MemberIDs)
	if err != nil {
		fail(c, http.StatusBadRequest, err.Error())
		return
	}
	success(c, conversation)
}

func (ctl *ConversationController) ListMessages(c *gin.Context) {
	user, ok := middleware.CurrentUser(c)
	if !ok {
		fail(c, http.StatusUnauthorized, "unauthorized")
		return
	}
	conversationID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		fail(c, http.StatusBadRequest, "invalid conversation id")
		return
	}
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	messages, err := ctl.conversations.ListMessages(user.ID, conversationID, limit)
	if err != nil {
		fail(c, http.StatusBadRequest, err.Error())
		return
	}
	success(c, messages)
}

func (ctl *ConversationController) SendMessage(c *gin.Context) {
	user, ok := middleware.CurrentUser(c)
	if !ok {
		fail(c, http.StatusUnauthorized, "unauthorized")
		return
	}
	conversationID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		fail(c, http.StatusBadRequest, "invalid conversation id")
		return
	}
	var req struct {
		Content string `json:"content" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, http.StatusBadRequest, err.Error())
		return
	}
	message, err := ctl.conversations.SendMessage(user.ID, conversationID, req.Content)
	if err != nil {
		fail(c, http.StatusBadRequest, err.Error())
		return
	}
	success(c, message)
}

func (ctl *ConversationController) MarkRead(c *gin.Context) {
	user, ok := middleware.CurrentUser(c)
	if !ok {
		fail(c, http.StatusUnauthorized, "unauthorized")
		return
	}
	conversationID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		fail(c, http.StatusBadRequest, "invalid conversation id")
		return
	}
	if err := ctl.conversations.MarkRead(user.ID, conversationID); err != nil {
		fail(c, http.StatusBadRequest, err.Error())
		return
	}
	success(c, gin.H{"conversation_id": conversationID, "status": "read"})
}

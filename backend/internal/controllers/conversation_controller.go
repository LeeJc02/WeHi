package controllers

import (
	"context"

	"github.com/LeeJc02/WeHi/backend/internal/app/presence"
	httpx "github.com/LeeJc02/WeHi/backend/internal/platform/httpx"
	"github.com/LeeJc02/WeHi/backend/pkg/contracts"

	"github.com/gin-gonic/gin"
)

type ConversationService interface {
	ListConversations(userID uint64) ([]contracts.ConversationDTO, error)
	EnsureDirectConversation(userID, targetID uint64) (*contracts.ConversationDTO, error)
	CreateGroupConversation(creatorID uint64, name string, memberIDs []uint64) (*contracts.ConversationDTO, error)
	RenameConversation(userID, conversationID uint64, name string) (*contracts.ConversationDTO, error)
	ListConversationMembers(userID, conversationID uint64, onlineUsers map[uint64]bool) ([]contracts.ConversationMemberDTO, error)
	AddConversationMembers(actorID, conversationID uint64, memberIDs []uint64) error
	RemoveConversationMember(actorID, conversationID, targetID uint64) error
	LeaveConversation(userID, conversationID uint64) error
	TransferOwnership(actorID, conversationID, targetID uint64) error
	UpdateSettings(userID, conversationID uint64, req contracts.UpdateConversationSettingsRequest) (*contracts.ConversationDTO, error)
}

type ConversationController struct {
	service  ConversationService
	presence *presence.Service
}

func NewConversationController(service ConversationService, presenceService *presence.Service) *ConversationController {
	return &ConversationController{service: service, presence: presenceService}
}

func (ctl *ConversationController) ListConversations(c *gin.Context) {
	user, _, ok := requireCurrentUser(c)
	if !ok {
		return
	}
	conversations, err := ctl.service.ListConversations(user.ID)
	if err != nil {
		httpx.FailError(c, err)
		return
	}
	httpx.Success(c, conversations)
}

func (ctl *ConversationController) CreateDirectConversation(c *gin.Context) {
	user, _, ok := requireCurrentUser(c)
	if !ok {
		return
	}
	var req contracts.CreateDirectConversationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Fail(c, 400, err.Error())
		return
	}
	conversation, err := ctl.service.EnsureDirectConversation(user.ID, req.TargetUserID)
	if err != nil {
		httpx.FailError(c, err)
		return
	}
	httpx.Success(c, conversation)
}

func (ctl *ConversationController) CreateGroupConversation(c *gin.Context) {
	user, _, ok := requireCurrentUser(c)
	if !ok {
		return
	}
	var req contracts.CreateGroupConversationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Fail(c, 400, err.Error())
		return
	}
	conversation, err := ctl.service.CreateGroupConversation(user.ID, req.Name, req.MemberIDs)
	if err != nil {
		httpx.FailError(c, err)
		return
	}
	httpx.Success(c, conversation)
}

func (ctl *ConversationController) RenameConversation(c *gin.Context) {
	user, _, ok := requireCurrentUser(c)
	if !ok {
		return
	}
	conversationID, err := parseUintParam(c, "id")
	if err != nil {
		httpx.Fail(c, 400, err.Error())
		return
	}
	var req contracts.RenameConversationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Fail(c, 400, err.Error())
		return
	}
	conversation, err := ctl.service.RenameConversation(user.ID, conversationID, req.Name)
	if err != nil {
		httpx.FailError(c, err)
		return
	}
	httpx.Success(c, conversation)
}

func (ctl *ConversationController) ListConversationMembers(c *gin.Context) {
	user, _, ok := requireCurrentUser(c)
	if !ok {
		return
	}
	conversationID, err := parseUintParam(c, "id")
	if err != nil {
		httpx.Fail(c, 400, err.Error())
		return
	}
	members, err := ctl.service.ListConversationMembers(user.ID, conversationID, ctl.onlineMembers(c.Request.Context()))
	if err != nil {
		httpx.FailError(c, err)
		return
	}
	httpx.Success(c, members)
}

func (ctl *ConversationController) AddConversationMembers(c *gin.Context) {
	user, _, ok := requireCurrentUser(c)
	if !ok {
		return
	}
	conversationID, err := parseUintParam(c, "id")
	if err != nil {
		httpx.Fail(c, 400, err.Error())
		return
	}
	var req contracts.AddConversationMembersRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Fail(c, 400, err.Error())
		return
	}
	if err := ctl.service.AddConversationMembers(user.ID, conversationID, req.MemberIDs); err != nil {
		httpx.FailError(c, err)
		return
	}
	httpx.Success(c, gin.H{"status": "members_added"})
}

func (ctl *ConversationController) RemoveConversationMember(c *gin.Context) {
	user, _, ok := requireCurrentUser(c)
	if !ok {
		return
	}
	conversationID, err := parseUintParam(c, "id")
	if err != nil {
		httpx.Fail(c, 400, err.Error())
		return
	}
	targetID, err := parseUintParam(c, "userId")
	if err != nil {
		httpx.Fail(c, 400, err.Error())
		return
	}
	if err := ctl.service.RemoveConversationMember(user.ID, conversationID, targetID); err != nil {
		httpx.FailError(c, err)
		return
	}
	httpx.Success(c, gin.H{"status": "member_removed"})
}

func (ctl *ConversationController) LeaveConversation(c *gin.Context) {
	user, _, ok := requireCurrentUser(c)
	if !ok {
		return
	}
	conversationID, err := parseUintParam(c, "id")
	if err != nil {
		httpx.Fail(c, 400, err.Error())
		return
	}
	if err := ctl.service.LeaveConversation(user.ID, conversationID); err != nil {
		httpx.FailError(c, err)
		return
	}
	httpx.Success(c, gin.H{"status": "left"})
}

func (ctl *ConversationController) TransferOwnership(c *gin.Context) {
	user, _, ok := requireCurrentUser(c)
	if !ok {
		return
	}
	conversationID, err := parseUintParam(c, "id")
	if err != nil {
		httpx.Fail(c, 400, err.Error())
		return
	}
	var req contracts.TransferOwnershipRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Fail(c, 400, err.Error())
		return
	}
	if err := ctl.service.TransferOwnership(user.ID, conversationID, req.UserID); err != nil {
		httpx.FailError(c, err)
		return
	}
	httpx.Success(c, gin.H{"status": "transferred"})
}

func (ctl *ConversationController) UpdateConversationSettings(c *gin.Context) {
	user, _, ok := requireCurrentUser(c)
	if !ok {
		return
	}
	conversationID, err := parseUintParam(c, "id")
	if err != nil {
		httpx.Fail(c, 400, err.Error())
		return
	}
	var req contracts.UpdateConversationSettingsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Fail(c, 400, err.Error())
		return
	}
	conversation, err := ctl.service.UpdateSettings(user.ID, conversationID, req)
	if err != nil {
		httpx.FailError(c, err)
		return
	}
	httpx.Success(c, conversation)
}

func (ctl *ConversationController) onlineMembers(ctx context.Context) map[uint64]bool {
	if ctl.presence == nil {
		return map[uint64]bool{}
	}
	onlineUsers, err := ctl.presence.OnlineUsers(ctx)
	if err != nil {
		return map[uint64]bool{}
	}
	onlineMap, err := ctl.presence.OnlineMap(ctx, onlineUsers)
	if err != nil {
		return map[uint64]bool{}
	}
	return onlineMap
}

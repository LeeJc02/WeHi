package controllers

import (
	httpx "awesomeproject/internal/platform/httpx"
	"awesomeproject/pkg/contracts"

	"github.com/gin-gonic/gin"
)

type FriendService interface {
	ListFriends(userID uint64) ([]contracts.FriendDTO, error)
	ListFriendRequests(userID uint64) ([]contracts.FriendRequestDTO, error)
	CreateFriendRequest(userID, addresseeID uint64, message string) (*contracts.FriendRequestDTO, error)
	ApproveFriendRequest(userID, requestID uint64) error
	RejectFriendRequest(userID, requestID uint64) error
}

type FriendController struct {
	service FriendService
}

func NewFriendController(service FriendService) *FriendController {
	return &FriendController{service: service}
}

func (ctl *FriendController) ListFriends(c *gin.Context) {
	user, _, ok := requireCurrentUser(c)
	if !ok {
		return
	}
	friends, err := ctl.service.ListFriends(user.ID)
	if err != nil {
		httpx.FailError(c, err)
		return
	}
	httpx.Success(c, friends)
}

func (ctl *FriendController) ListFriendRequests(c *gin.Context) {
	user, _, ok := requireCurrentUser(c)
	if !ok {
		return
	}
	requests, err := ctl.service.ListFriendRequests(user.ID)
	if err != nil {
		httpx.FailError(c, err)
		return
	}
	httpx.Success(c, requests)
}

func (ctl *FriendController) CreateFriendRequest(c *gin.Context) {
	user, _, ok := requireCurrentUser(c)
	if !ok {
		return
	}
	var req contracts.CreateFriendRequestRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Fail(c, 400, err.Error())
		return
	}
	request, err := ctl.service.CreateFriendRequest(user.ID, req.AddresseeID, req.Message)
	if err != nil {
		httpx.FailError(c, err)
		return
	}
	httpx.Success(c, request)
}

func (ctl *FriendController) ApproveFriendRequest(c *gin.Context) {
	user, _, ok := requireCurrentUser(c)
	if !ok {
		return
	}
	requestID, err := parseUintParam(c, "id")
	if err != nil {
		httpx.Fail(c, 400, err.Error())
		return
	}
	if err := ctl.service.ApproveFriendRequest(user.ID, requestID); err != nil {
		httpx.FailError(c, err)
		return
	}
	httpx.Success(c, gin.H{"status": "accepted"})
}

func (ctl *FriendController) RejectFriendRequest(c *gin.Context) {
	user, _, ok := requireCurrentUser(c)
	if !ok {
		return
	}
	requestID, err := parseUintParam(c, "id")
	if err != nil {
		httpx.Fail(c, 400, err.Error())
		return
	}
	if err := ctl.service.RejectFriendRequest(user.ID, requestID); err != nil {
		httpx.FailError(c, err)
		return
	}
	httpx.Success(c, gin.H{"status": "rejected"})
}

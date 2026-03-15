package controllers

import (
	"net/http"

	"awesomeproject/backend/internal/middleware"
	"awesomeproject/backend/internal/services"

	"github.com/gin-gonic/gin"
)

type FriendController struct {
	friends *services.FriendService
}

func NewFriendController(friends *services.FriendService) *FriendController {
	return &FriendController{friends: friends}
}

func (ctl *FriendController) Create(c *gin.Context) {
	user, ok := middleware.CurrentUser(c)
	if !ok {
		fail(c, http.StatusUnauthorized, "unauthorized")
		return
	}
	var req struct {
		FriendID uint64 `json:"friend_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, http.StatusBadRequest, err.Error())
		return
	}
	if err := ctl.friends.AddFriend(user.ID, req.FriendID); err != nil {
		fail(c, http.StatusBadRequest, err.Error())
		return
	}
	success(c, gin.H{"friend_id": req.FriendID})
}

func (ctl *FriendController) List(c *gin.Context) {
	user, ok := middleware.CurrentUser(c)
	if !ok {
		fail(c, http.StatusUnauthorized, "unauthorized")
		return
	}
	friends, err := ctl.friends.ListFriends(user.ID)
	if err != nil {
		fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	success(c, friends)
}

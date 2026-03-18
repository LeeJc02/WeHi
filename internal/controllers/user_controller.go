package controllers

import (
	httpx "awesomeproject/internal/platform/httpx"
	"awesomeproject/pkg/contracts"

	"github.com/gin-gonic/gin"
)

type UserReader interface {
	ListUsers(currentUserID uint64) ([]contracts.UserProfile, error)
	UpdateProfile(userID uint64, displayName string) (*contracts.UserProfile, error)
}

type UserController struct {
	service UserReader
}

func NewUserController(service UserReader) *UserController {
	return &UserController{service: service}
}

func (ctl *UserController) GetMe(c *gin.Context) {
	user, _, ok := requireCurrentUser(c)
	if !ok {
		return
	}
	httpx.Success(c, gin.H{"id": user.ID, "username": user.Username, "display_name": user.DisplayName})
}

func (ctl *UserController) ListUsers(c *gin.Context) {
	user, _, ok := requireCurrentUser(c)
	if !ok {
		return
	}
	users, err := ctl.service.ListUsers(user.ID)
	if err != nil {
		httpx.FailError(c, err)
		return
	}
	httpx.Success(c, users)
}

func (ctl *UserController) UpdateMe(c *gin.Context) {
	user, _, ok := requireCurrentUser(c)
	if !ok {
		return
	}
	var req contracts.UpdateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Fail(c, 400, err.Error())
		return
	}
	profile, err := ctl.service.UpdateProfile(user.ID, req.DisplayName)
	if err != nil {
		httpx.FailError(c, err)
		return
	}
	httpx.Success(c, profile)
}

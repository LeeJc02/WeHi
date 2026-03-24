package controllers

import (
	"github.com/LeeJc02/WeHi/backend/internal/app/admin"
	httpx "github.com/LeeJc02/WeHi/backend/internal/platform/httpx"
	"github.com/LeeJc02/WeHi/backend/pkg/contracts"

	"github.com/gin-gonic/gin"
)

type AdminAuthService interface {
	Login(username, password string) (*contracts.AdminAuthPayload, error)
	Me(adminID uint64) (*contracts.AdminProfile, error)
	ChangePassword(adminID uint64, currentPassword, newPassword string) error
}

type AdminAuthController struct {
	service AdminAuthService
}

func NewAdminAuthController(service AdminAuthService) *AdminAuthController {
	return &AdminAuthController{service: service}
}

func (ctl *AdminAuthController) Login(c *gin.Context) {
	var req contracts.AdminLoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Fail(c, 400, err.Error())
		return
	}
	payload, err := ctl.service.Login(req.Username, req.Password)
	if err != nil {
		httpx.FailError(c, err)
		return
	}
	httpx.Success(c, payload)
}

func (ctl *AdminAuthController) Me(c *gin.Context) {
	adminUser, ok := admin.CurrentAdmin(c)
	if !ok {
		httpx.Fail(c, 401, "admin not authenticated")
		return
	}
	profile, err := ctl.service.Me(adminUser.ID)
	if err != nil {
		httpx.FailError(c, err)
		return
	}
	httpx.Success(c, profile)
}

func (ctl *AdminAuthController) ChangePassword(c *gin.Context) {
	adminUser, ok := admin.CurrentAdmin(c)
	if !ok {
		httpx.Fail(c, 401, "admin not authenticated")
		return
	}
	var req contracts.AdminChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Fail(c, 400, err.Error())
		return
	}
	if err := ctl.service.ChangePassword(adminUser.ID, req.CurrentPassword, req.NewPassword); err != nil {
		httpx.FailError(c, err)
		return
	}
	httpx.Success(c, gin.H{"status": "password_changed"})
}

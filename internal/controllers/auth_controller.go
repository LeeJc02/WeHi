package controllers

import (
	"awesomeproject/internal/app/auth"
	httpx "awesomeproject/internal/platform/httpx"
	"awesomeproject/pkg/contracts"

	"github.com/gin-gonic/gin"
)

type AuthController struct {
	service *auth.Service
}

func NewAuthController(service *auth.Service) *AuthController {
	return &AuthController{service: service}
}

func (ctl *AuthController) Register(c *gin.Context) {
	var req contracts.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Fail(c, 400, err.Error())
		return
	}
	user, err := ctl.service.Register(req.Username, req.DisplayName, req.Password)
	if err != nil {
		httpx.FailError(c, err)
		return
	}
	httpx.Success(c, user)
}

func (ctl *AuthController) Login(c *gin.Context) {
	var req contracts.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Fail(c, 400, err.Error())
		return
	}
	payload, err := ctl.service.Login(
		c.Request.Context(),
		req.Username,
		req.Password,
		c.GetHeader("X-Device-Id"),
		c.Request.UserAgent(),
	)
	if err != nil {
		httpx.FailError(c, err)
		return
	}
	httpx.Success(c, payload)
}

func (ctl *AuthController) Refresh(c *gin.Context) {
	var req contracts.RefreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Fail(c, 400, err.Error())
		return
	}
	payload, err := ctl.service.Refresh(c.Request.Context(), req.RefreshToken)
	if err != nil {
		httpx.FailError(c, err)
		return
	}
	httpx.Success(c, payload)
}

func (ctl *AuthController) ListSessions(c *gin.Context) {
	user, sessionID, ok := requireCurrentUser(c)
	if !ok {
		return
	}
	sessions, err := ctl.service.ListSessions(c.Request.Context(), user.ID, sessionID)
	if err != nil {
		httpx.FailError(c, err)
		return
	}
	httpx.Success(c, sessions)
}

func (ctl *AuthController) Logout(c *gin.Context) {
	var req contracts.LogoutRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Fail(c, 400, err.Error())
		return
	}
	if err := ctl.service.Logout(c.Request.Context(), req.RefreshToken); err != nil {
		httpx.FailError(c, err)
		return
	}
	httpx.Success(c, gin.H{"status": "logged_out"})
}

func (ctl *AuthController) LogoutAll(c *gin.Context) {
	user, _, ok := requireCurrentUser(c)
	if !ok {
		return
	}
	if err := ctl.service.LogoutAll(c.Request.Context(), user.ID); err != nil {
		httpx.FailError(c, err)
		return
	}
	httpx.Success(c, gin.H{"status": "all_sessions_revoked"})
}

func (ctl *AuthController) LogoutOthers(c *gin.Context) {
	user, sessionID, ok := requireCurrentUser(c)
	if !ok {
		return
	}
	if err := ctl.service.LogoutOthers(c.Request.Context(), user.ID, sessionID); err != nil {
		httpx.FailError(c, err)
		return
	}
	httpx.Success(c, gin.H{"status": "other_sessions_revoked"})
}

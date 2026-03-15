package controllers

import (
	"net/http"

	"awesomeproject/backend/internal/middleware"
	"awesomeproject/backend/internal/services"

	"github.com/gin-gonic/gin"
)

type AuthController struct {
	auth *services.AuthService
}

func NewAuthController(auth *services.AuthService) *AuthController {
	return &AuthController{auth: auth}
}

func (ctl *AuthController) Register(c *gin.Context) {
	var req struct {
		Username    string `json:"username" binding:"required"`
		DisplayName string `json:"display_name" binding:"required"`
		Password    string `json:"password" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, http.StatusBadRequest, err.Error())
		return
	}
	user, err := ctl.auth.Register(req.Username, req.DisplayName, req.Password)
	if err != nil {
		fail(c, http.StatusBadRequest, err.Error())
		return
	}
	success(c, user)
}

func (ctl *AuthController) Login(c *gin.Context) {
	var req struct {
		Username string `json:"username" binding:"required"`
		Password string `json:"password" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, http.StatusBadRequest, err.Error())
		return
	}

	session, user, err := ctl.auth.Login(req.Username, req.Password)
	if err != nil {
		fail(c, http.StatusUnauthorized, err.Error())
		return
	}
	success(c, gin.H{
		"token": session.Token,
		"user":  user,
	})
}

func (ctl *AuthController) Me(c *gin.Context) {
	user, ok := middleware.CurrentUser(c)
	if !ok {
		fail(c, http.StatusUnauthorized, "unauthorized")
		return
	}
	success(c, user)
}

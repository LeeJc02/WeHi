package controllers

import (
	"net/http"

	"awesomeproject/backend/internal/repositories"

	"github.com/gin-gonic/gin"
)

type UserController struct {
	users *repositories.UserRepository
}

func NewUserController(users *repositories.UserRepository) *UserController {
	return &UserController{users: users}
}

func (ctl *UserController) List(c *gin.Context) {
	users, err := ctl.users.List()
	if err != nil {
		fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	success(c, users)
}

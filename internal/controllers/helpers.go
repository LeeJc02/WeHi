package controllers

import (
	"net/http"
	"strconv"

	"awesomeproject/internal/app/auth"
	"awesomeproject/internal/app/repository"
	httpx "awesomeproject/internal/platform/httpx"

	"github.com/gin-gonic/gin"
)

func parseUintParam(c *gin.Context, key string) (uint64, error) {
	value, err := strconv.ParseUint(c.Param(key), 10, 64)
	if err != nil {
		return 0, err
	}
	return value, nil
}

func requireCurrentUser(c *gin.Context) (*repository.User, string, bool) {
	user, sessionID, ok := auth.CurrentUser(c)
	if !ok {
		httpx.Fail(c, http.StatusUnauthorized, "unauthorized")
		return nil, "", false
	}
	return user, sessionID, true
}

package middleware

import (
	"net/http"
	"strings"

	"awesomeproject/backend/internal/models"
	"awesomeproject/backend/internal/services"

	"github.com/gin-gonic/gin"
)

const currentUserKey = "current_user"

func Auth(authService *services.AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := strings.TrimSpace(c.GetHeader("Authorization"))
		if header == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"code": http.StatusUnauthorized, "message": "missing Authorization header"})
			c.Abort()
			return
		}
		parts := strings.SplitN(header, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			c.JSON(http.StatusUnauthorized, gin.H{"code": http.StatusUnauthorized, "message": "invalid Authorization header"})
			c.Abort()
			return
		}
		user, err := authService.Authenticate(parts[1])
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"code": http.StatusUnauthorized, "message": err.Error()})
			c.Abort()
			return
		}
		c.Set(currentUserKey, user)
		c.Next()
	}
}

func CurrentUser(c *gin.Context) (*models.User, bool) {
	value, ok := c.Get(currentUserKey)
	if !ok {
		return nil, false
	}
	user, ok := value.(*models.User)
	return user, ok
}

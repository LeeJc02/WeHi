package routes

import (
	"context"
	"time"

	"awesomeproject/internal/app/auth"
	"awesomeproject/internal/config"
	"awesomeproject/internal/controllers"
	"awesomeproject/internal/platform/db"
	httpx "awesomeproject/internal/platform/httpx"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

func RegisterAuthRoutes(router *gin.Engine, cfg config.Config, gormDB *gorm.DB, redis *redis.Client, authService *auth.Service, controller *controllers.AuthController) {
	router.GET("/health", func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
		defer cancel()
		dbStatus := "up"
		redisStatus := "up"
		if err := db.Ping(gormDB); err != nil {
			dbStatus = err.Error()
		}
		if err := redis.Ping(ctx).Err(); err != nil {
			redisStatus = err.Error()
		}
		httpx.Success(c, gin.H{
			"service":  gin.H{"name": cfg.ServiceName, "status": "up"},
			"database": gin.H{"status": dbStatus},
			"redis":    gin.H{"status": redisStatus},
		})
	})

	api := router.Group("/api/v1/auth")
	api.POST("/register", controller.Register)
	api.POST("/login", controller.Login)
	api.POST("/refresh", controller.Refresh)

	secured := api.Group("")
	secured.Use(authService.Middleware())
	secured.GET("/sessions", controller.ListSessions)
	secured.POST("/logout", controller.Logout)
	secured.POST("/logout-all", controller.LogoutAll)
}

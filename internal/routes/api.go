package routes

import (
	"context"
	"net/http"
	"time"

	"awesomeproject/internal/app/auth"
	"awesomeproject/internal/config"
	"awesomeproject/internal/controllers"
	"awesomeproject/internal/platform/db"
	httpx "awesomeproject/internal/platform/httpx"
	"awesomeproject/internal/platform/search"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

func RegisterAPIRoutes(router *gin.Engine, cfg config.Config, gormDB *gorm.DB, redis *redis.Client, searchClient *search.Client, authService *auth.Service, userController *controllers.UserController, friendController *controllers.FriendController, conversationController *controllers.ConversationController, messageController *controllers.MessageController, searchController *controllers.SearchController, syncController *controllers.SyncController) {
	router.GET("/health", func(c *gin.Context) {
		httpx.Success(c, gin.H{"service": gin.H{"name": cfg.ServiceName, "status": "up"}})
	})
	router.GET("/api/v1/system/ready", func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
		defer cancel()
		payload := gin.H{
			"service":       gin.H{"name": cfg.ServiceName, "status": "up"},
			"database":      gin.H{"status": "up"},
			"redis":         gin.H{"status": "up"},
			"rabbitmq":      gin.H{"status": "up"},
			"elasticsearch": gin.H{"status": "up"},
		}
		status := http.StatusOK
		if err := db.Ping(gormDB); err != nil {
			status = http.StatusServiceUnavailable
			payload["database"] = gin.H{"status": "down", "error": err.Error()}
		}
		if err := redis.Ping(ctx).Err(); err != nil {
			status = http.StatusServiceUnavailable
			payload["redis"] = gin.H{"status": "down", "error": err.Error()}
		}
		if err := searchClient.Ping(ctx); err != nil {
			status = http.StatusServiceUnavailable
			payload["elasticsearch"] = gin.H{"status": "down", "error": err.Error()}
		}
		c.JSON(status, gin.H{"code": 0, "message": "ok", "data": payload})
	})

	api := router.Group("/api/v1")
	api.Use(authService.Middleware())

	api.GET("/users/me", userController.GetMe)
	api.PATCH("/users/me", userController.UpdateMe)
	api.GET("/users", userController.ListUsers)
	api.GET("/friends", friendController.ListFriends)
	api.GET("/friend-requests", friendController.ListFriendRequests)
	api.POST("/friend-requests", friendController.CreateFriendRequest)
	api.POST("/friend-requests/:id/approve", friendController.ApproveFriendRequest)
	api.POST("/friend-requests/:id/reject", friendController.RejectFriendRequest)

	api.GET("/conversations", conversationController.ListConversations)
	api.POST("/conversations/direct", conversationController.CreateDirectConversation)
	api.POST("/conversations/group", conversationController.CreateGroupConversation)
	api.PATCH("/conversations/:id", conversationController.RenameConversation)
	api.GET("/conversations/:id/members", conversationController.ListConversationMembers)
	api.POST("/conversations/:id/members", conversationController.AddConversationMembers)
	api.DELETE("/conversations/:id/members/:userId", conversationController.RemoveConversationMember)
	api.POST("/conversations/:id/leave", conversationController.LeaveConversation)
	api.POST("/conversations/:id/transfer-owner", conversationController.TransferOwnership)
	api.POST("/conversations/:id/pin", conversationController.SetConversationPin)
	api.GET("/conversations/:id/messages", messageController.ListMessages)
	api.POST("/conversations/:id/messages", messageController.SendMessage)
	api.POST("/conversations/:id/read", messageController.MarkRead)
	api.GET("/search", searchController.Search)
	api.GET("/sync/cursor", syncController.Cursor)
	api.GET("/sync/events", syncController.Events)
}

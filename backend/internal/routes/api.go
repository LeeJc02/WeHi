package routes

import (
	"context"
	"net/http"
	"time"

	"github.com/LeeJc02/WeHi/backend/internal/app/admin"
	"github.com/LeeJc02/WeHi/backend/internal/app/auth"
	"github.com/LeeJc02/WeHi/backend/internal/config"
	"github.com/LeeJc02/WeHi/backend/internal/controllers"
	"github.com/LeeJc02/WeHi/backend/internal/platform/db"
	httpx "github.com/LeeJc02/WeHi/backend/internal/platform/httpx"
	"github.com/LeeJc02/WeHi/backend/internal/platform/search"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

func RegisterAPIRoutes(router *gin.Engine, cfg config.Config, gormDB *gorm.DB, redis *redis.Client, searchClient *search.Client, authService *auth.Service, adminService *admin.Service, adminAuthController *controllers.AdminAuthController, adminAIController *controllers.AdminAIController, adminMonitorController *controllers.AdminMonitorController, adminAuditController *controllers.AdminAuditController, adminDiagnosticsController *controllers.AdminDiagnosticsController, adminMaintenanceController *controllers.AdminMaintenanceController, userController *controllers.UserController, friendController *controllers.FriendController, conversationController *controllers.ConversationController, messageController *controllers.MessageController, uploadController *controllers.UploadController, searchController *controllers.SearchController, syncController *controllers.SyncController) {
	router.GET("/health", func(c *gin.Context) {
		httpx.Success(c, gin.H{"service": gin.H{"name": cfg.ServiceName, "status": "up"}})
	})
	router.GET("/uploads/:key", uploadController.Download)
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

	adminAuth := router.Group("/api/v1/admin/auth")
	adminAuth.POST("/login", adminAuthController.Login)
	adminAuthSecured := adminAuth.Group("")
	adminAuthSecured.Use(adminService.Middleware(false))
	adminAuthSecured.GET("/me", adminAuthController.Me)
	adminAuthSecured.POST("/change-password", adminAuthController.ChangePassword)
	adminAPI := router.Group("/api/v1/admin")
	adminAPI.Use(adminService.Middleware(true))
	adminAPI.GET("/ai-config", adminAIController.GetConfig)
	adminAPI.PUT("/ai-config", adminAIController.UpdateConfig)
	adminAPI.GET("/ai/retry-jobs", adminAIController.ListRetryJobs)
	adminAPI.GET("/ai/retry-jobs/:id", adminAIController.RetryJobDetail)
	adminAPI.POST("/ai/retry-jobs/:id/retry-now", adminAIController.RetryJobNow)
	adminAPI.POST("/ai/retry-jobs/retry-batch", adminAIController.RetryJobs)
	adminAPI.POST("/ai/retry-jobs/cleanup", adminAIController.CleanupRetryJobs)
	adminAPI.GET("/monitor/overview", adminMonitorController.Overview)
	adminAPI.GET("/monitor/timeseries", adminMonitorController.Timeseries)
	adminAPI.GET("/audit/ai-calls", adminAuditController.List)
	adminAPI.GET("/audit/ai-calls/:id", adminAuditController.Detail)
	adminAPI.GET("/messages/resolve", adminDiagnosticsController.ResolveMessage)
	adminAPI.GET("/message-journey/:messageId", adminDiagnosticsController.MessageJourney)
	adminAPI.GET("/conversations/:id/consistency", adminDiagnosticsController.ConversationConsistency)
	adminAPI.GET("/conversations/:id/events", adminDiagnosticsController.ConversationEvents)
	adminAPI.POST("/search/reindex", adminMaintenanceController.ReindexSearch)

	api.GET("/users/me", userController.GetMe)
	api.PATCH("/users/me", userController.UpdateMe)
	api.GET("/users", userController.ListUsers)
	api.GET("/friends", friendController.ListFriends)
	api.PATCH("/friends/:id/remark", friendController.UpdateRemark)
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
	api.PATCH("/conversations/:id/settings", conversationController.UpdateConversationSettings)
	api.GET("/conversations/:id/messages", messageController.ListMessages)
	api.POST("/conversations/:id/messages", messageController.SendMessage)
	api.POST("/conversations/:id/typing", messageController.UpdateTyping)
	api.POST("/conversations/:id/read", messageController.MarkRead)
	api.POST("/messages/:id/recall", messageController.Recall)
	api.POST("/uploads/presign", uploadController.Presign)
	api.PUT("/uploads/object/:key", uploadController.PutObject)
	api.POST("/uploads/complete", uploadController.Complete)
	api.GET("/search", searchController.Search)
	api.GET("/sync/cursor", syncController.Cursor)
	api.GET("/sync/events", syncController.Events)
}

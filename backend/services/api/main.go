package main

import (
	"context"
	"log"

	adminapp "github.com/LeeJc02/WeHi/backend/internal/app/admin"
	"github.com/LeeJc02/WeHi/backend/internal/app/ai"
	"github.com/LeeJc02/WeHi/backend/internal/app/auth"
	"github.com/LeeJc02/WeHi/backend/internal/app/chat"
	"github.com/LeeJc02/WeHi/backend/internal/app/presence"
	"github.com/LeeJc02/WeHi/backend/internal/app/repository"
	syncapp "github.com/LeeJc02/WeHi/backend/internal/app/sync"
	"github.com/LeeJc02/WeHi/backend/internal/app/upload"
	"github.com/LeeJc02/WeHi/backend/internal/config"
	"github.com/LeeJc02/WeHi/backend/internal/controllers"
	"github.com/LeeJc02/WeHi/backend/internal/platform/db"
	httpx "github.com/LeeJc02/WeHi/backend/internal/platform/httpx"
	"github.com/LeeJc02/WeHi/backend/internal/platform/observability"
	"github.com/LeeJc02/WeHi/backend/internal/platform/rabbit"
	redisclient "github.com/LeeJc02/WeHi/backend/internal/platform/redis"
	"github.com/LeeJc02/WeHi/backend/internal/platform/search"
	"github.com/LeeJc02/WeHi/backend/internal/routes"

	"github.com/gin-gonic/gin"
)

func main() {
	cfg := config.Load("api-service", "8082")
	shutdownTracing, err := observability.Init(context.Background(), cfg)
	if err != nil {
		log.Fatal(err)
	}
	defer func() { _ = shutdownTracing(context.Background()) }()
	gormDB, err := db.OpenGorm(cfg.MySQLDSN, cfg.ServiceName)
	if err != nil {
		log.Fatal(err)
	}
	redis := redisclient.New(cfg.RedisAddr, cfg.RedisPass)
	rabbitClient, err := rabbit.New(cfg.RabbitURL, cfg.RabbitExch)
	if err != nil {
		log.Fatal(err)
	}
	defer rabbitClient.Close()

	searchClient := search.New(cfg.ElasticsearchURL)
	repo := repository.New(gormDB)
	authService := auth.NewService(repo, redis, cfg)
	adminService := adminapp.NewService(repo, cfg)
	if err := adminService.EnsureSeed(); err != nil {
		log.Fatal(err)
	}
	aiConfigService := adminapp.NewAIConfigService(cfg.AIConfigPath)
	if err := aiConfigService.EnsureFile(); err != nil {
		log.Fatal(err)
	}
	aiService := ai.NewService(repo, aiConfigService)
	aiService.Start(context.Background())
	monitorService := adminapp.NewMonitorService(cfg)
	monitorService.Start(context.Background())
	presenceService := presence.NewService(redis)
	diagnosticsService := adminapp.NewDiagnosticsService(repo, presenceService)
	chatServices := chat.NewServices(repo, rabbitClient, searchClient, presenceService, aiService, cfg.ElasticsearchMessagesIndex, cfg.ElasticsearchConversationsIndex)
	aiService.SetReplyHandler(func(conversationID, botUserID uint64, messageType, content string) error {
		return chatServices.Message.EmitInternalMessage(botUserID, conversationID, messageType, content)
	})
	uploadService := upload.NewService(cfg.UploadsDir)
	if err := uploadService.EnsureReady(); err != nil {
		log.Fatal(err)
	}
	userController := controllers.NewUserController(chatServices.User)
	adminAuthController := controllers.NewAdminAuthController(adminService)
	adminAIController := controllers.NewAdminAIController(aiConfigService, aiService)
	adminMonitorController := controllers.NewAdminMonitorController(monitorService)
	adminAuditController := controllers.NewAdminAuditController(aiService)
	adminDiagnosticsController := controllers.NewAdminDiagnosticsController(diagnosticsService)
	adminMaintenanceController := controllers.NewAdminMaintenanceController(chatServices.Search)
	friendController := controllers.NewFriendController(chatServices.Friend)
	conversationController := controllers.NewConversationController(chatServices.Conversation, presenceService)
	messageController := controllers.NewMessageController(chatServices.Message)
	uploadController := controllers.NewUploadController(uploadService)
	searchController := controllers.NewSearchController(chatServices.Search)
	syncController := controllers.NewSyncController(syncapp.NewService(repo))

	router := gin.New()
	router.Use(httpx.RequestID(), observability.GinMiddleware(cfg.ServiceName), httpx.StructuredLogger(cfg.ServiceName), httpx.Metrics(cfg.ServiceName), gin.Recovery(), httpx.CORS(cfg.CORSOrigins))
	router.GET("/metrics", httpx.MetricsHandler())
	routes.RegisterAPIRoutes(router, cfg, gormDB, redis, searchClient, authService, adminService, adminAuthController, adminAIController, adminMonitorController, adminAuditController, adminDiagnosticsController, adminMaintenanceController, userController, friendController, conversationController, messageController, uploadController, searchController, syncController)

	if err := router.Run(":" + cfg.AppPort); err != nil {
		log.Fatal(err)
	}
}

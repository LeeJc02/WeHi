package main

import (
	"log"

	"awesomeproject/internal/app/auth"
	"awesomeproject/internal/app/chat"
	"awesomeproject/internal/app/presence"
	"awesomeproject/internal/app/repository"
	syncapp "awesomeproject/internal/app/sync"
	"awesomeproject/internal/config"
	"awesomeproject/internal/controllers"
	"awesomeproject/internal/platform/db"
	httpx "awesomeproject/internal/platform/httpx"
	"awesomeproject/internal/platform/rabbit"
	redisclient "awesomeproject/internal/platform/redis"
	"awesomeproject/internal/platform/search"
	"awesomeproject/internal/routes"

	"github.com/gin-gonic/gin"
)

func main() {
	cfg := config.Load("api-service", "8082")
	gormDB, err := db.OpenGorm(cfg.MySQLDSN)
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
	presenceService := presence.NewService(redis)
	chatServices := chat.NewServices(repo, rabbitClient, searchClient, cfg.ElasticsearchMessagesIndex, cfg.ElasticsearchConversationsIndex)
	userController := controllers.NewUserController(chatServices.User)
	friendController := controllers.NewFriendController(chatServices.Friend)
	conversationController := controllers.NewConversationController(chatServices.Conversation, presenceService)
	messageController := controllers.NewMessageController(chatServices.Message)
	searchController := controllers.NewSearchController(chatServices.Search)
	syncController := controllers.NewSyncController(syncapp.NewService(repo))

	router := gin.New()
	router.Use(httpx.RequestID(), httpx.StructuredLogger(cfg.ServiceName), httpx.Metrics(cfg.ServiceName), gin.Recovery(), httpx.CORS(cfg.CORSOrigins))
	router.GET("/metrics", httpx.MetricsHandler())
	routes.RegisterAPIRoutes(router, cfg, gormDB, redis, searchClient, authService, userController, friendController, conversationController, messageController, searchController, syncController)

	if err := router.Run(":" + cfg.AppPort); err != nil {
		log.Fatal(err)
	}
}

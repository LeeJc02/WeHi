package routes

import (
	"net/http"

	"awesomeproject/backend/internal/config"
	"awesomeproject/backend/internal/controllers"
	"awesomeproject/backend/internal/middleware"
	"awesomeproject/backend/internal/repositories"
	"awesomeproject/backend/internal/services"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func NewRouter(db *gorm.DB, cfg config.Config) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(gin.Logger(), gin.Recovery(), cors(cfg.FrontendOrigin))

	userRepository := repositories.NewUserRepository(db)
	sessionRepository := repositories.NewSessionRepository(db)
	friendRepository := repositories.NewFriendRepository(db)
	conversationRepository := repositories.NewConversationRepository(db)
	messageRepository := repositories.NewMessageRepository(db)

	authService := services.NewAuthService(userRepository, sessionRepository)
	friendService := services.NewFriendService(userRepository, friendRepository)
	conversationService := services.NewConversationService(userRepository, conversationRepository, messageRepository)

	authController := controllers.NewAuthController(authService)
	userController := controllers.NewUserController(userRepository)
	friendController := controllers.NewFriendController(friendService)
	conversationController := controllers.NewConversationController(conversationService)
	docsController := controllers.NewDocsController()

	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"code": 0, "message": "ok", "data": gin.H{"service": "chat-mvc-backend", "status": "up"}})
	})
	router.GET("/openapi.yaml", docsController.OpenAPI)

	api := router.Group("/api/v1")
	{
		api.POST("/auth/register", authController.Register)
		api.POST("/auth/login", authController.Login)
	}

	authorized := api.Group("")
	authorized.Use(middleware.Auth(authService))
	{
		authorized.GET("/users", userController.List)
		authorized.GET("/users/me", authController.Me)
		authorized.GET("/friends", friendController.List)
		authorized.POST("/friends", friendController.Create)
		authorized.GET("/conversations", conversationController.List)
		authorized.POST("/conversations/direct", conversationController.CreateDirect)
		authorized.POST("/conversations/group", conversationController.CreateGroup)
		authorized.GET("/conversations/:id/messages", conversationController.ListMessages)
		authorized.POST("/conversations/:id/messages", conversationController.SendMessage)
		authorized.POST("/conversations/:id/read", conversationController.MarkRead)
	}

	return router
}

func cors(frontendOrigin string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", frontendOrigin)
		c.Header("Access-Control-Allow-Methods", "GET,POST,OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Authorization,Content-Type")
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}

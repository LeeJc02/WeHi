package main

import (
	"context"
	"log"

	"github.com/LeeJc02/WeHi/backend/internal/app/auth"
	"github.com/LeeJc02/WeHi/backend/internal/app/repository"
	"github.com/LeeJc02/WeHi/backend/internal/config"
	"github.com/LeeJc02/WeHi/backend/internal/controllers"
	"github.com/LeeJc02/WeHi/backend/internal/platform/db"
	httpx "github.com/LeeJc02/WeHi/backend/internal/platform/httpx"
	"github.com/LeeJc02/WeHi/backend/internal/platform/observability"
	redisclient "github.com/LeeJc02/WeHi/backend/internal/platform/redis"
	"github.com/LeeJc02/WeHi/backend/internal/routes"

	"github.com/gin-gonic/gin"
)

func main() {
	cfg := config.Load("auth-service", "8081")
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
	repo := repository.New(gormDB)
	authService := auth.NewService(repo, redis, cfg)
	authController := controllers.NewAuthController(authService)

	router := gin.New()
	router.Use(httpx.RequestID(), observability.GinMiddleware(cfg.ServiceName), httpx.StructuredLogger(cfg.ServiceName), httpx.Metrics(cfg.ServiceName), gin.Recovery(), httpx.CORS(cfg.CORSOrigins))
	router.GET("/metrics", httpx.MetricsHandler())
	routes.RegisterAuthRoutes(router, cfg, gormDB, redis, authService, authController)

	if err := router.Run(":" + cfg.AppPort); err != nil {
		log.Fatal(err)
	}
}

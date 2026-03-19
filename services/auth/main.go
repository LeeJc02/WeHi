package main

import (
	"context"
	"log"

	"awesomeproject/internal/app/auth"
	"awesomeproject/internal/app/repository"
	"awesomeproject/internal/config"
	"awesomeproject/internal/controllers"
	"awesomeproject/internal/platform/db"
	httpx "awesomeproject/internal/platform/httpx"
	"awesomeproject/internal/platform/observability"
	redisclient "awesomeproject/internal/platform/redis"
	"awesomeproject/internal/routes"

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

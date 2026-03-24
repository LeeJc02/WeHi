package routes

import (
	httpx "github.com/LeeJc02/WeHi/backend/internal/platform/httpx"

	"github.com/LeeJc02/WeHi/backend/internal/controllers"

	"github.com/gin-gonic/gin"
)

func RegisterRealtimeRoutes(router *gin.Engine, controller *controllers.RealtimeController) {
	router.GET("/health", func(c *gin.Context) {
		httpx.Success(c, gin.H{"service": gin.H{"name": "realtime-service", "status": "up"}})
	})
	router.GET("/ws", controller.ServeWS)
}

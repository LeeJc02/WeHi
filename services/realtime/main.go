package main

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"awesomeproject/internal/app/auth"
	"awesomeproject/internal/app/chat"
	"awesomeproject/internal/app/presence"
	"awesomeproject/internal/app/repository"
	"awesomeproject/internal/config"
	"awesomeproject/internal/controllers"
	"awesomeproject/internal/platform/db"
	httpx "awesomeproject/internal/platform/httpx"
	"awesomeproject/internal/platform/rabbit"
	redisclient "awesomeproject/internal/platform/redis"
	"awesomeproject/internal/platform/search"
	"awesomeproject/internal/realtime"
	"awesomeproject/internal/routes"
	"awesomeproject/pkg/contracts"

	"github.com/gin-gonic/gin"
)

func main() {
	cfg := config.Load("realtime-service", "8083")
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

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	_ = searchClient.EnsureIndex(ctx, cfg.ElasticsearchMessagesIndex, map[string]any{
		"mappings": map[string]any{
			"properties": map[string]any{
				"message_id":        map[string]any{"type": "unsigned_long"},
				"conversation_id":   map[string]any{"type": "unsigned_long"},
				"conversation_name": map[string]any{"type": "text"},
				"sender_id":         map[string]any{"type": "unsigned_long"},
				"message_type":      map[string]any{"type": "keyword"},
				"content":           map[string]any{"type": "text"},
				"created_at":        map[string]any{"type": "date"},
			},
		},
	})
	_ = searchClient.EnsureIndex(ctx, cfg.ElasticsearchConversationsIndex, map[string]any{
		"mappings": map[string]any{
			"properties": map[string]any{
				"conversation_id": map[string]any{"type": "unsigned_long"},
				"name":            map[string]any{"type": "text"},
				"type":            map[string]any{"type": "keyword"},
				"updated_at":      map[string]any{"type": "date"},
			},
		},
	})

	h := realtime.NewHub()
	realtimeController := controllers.NewRealtimeController(authService, presenceService, h)
	if err := rabbitClient.Consume("realtime.events", []string{"message.new", "conversation.read", "friend.request", "sync.notify"}, func(routingKey string, body []byte) error {
		switch routingKey {
		case "message.new":
			var event contracts.MessageFanoutEvent
			if err := json.Unmarshal(body, &event); err != nil {
				return err
			}
			h.Broadcast(event.Recipients, contracts.EventEnvelope{Type: routingKey, Payload: event})
		case "conversation.read":
			var event contracts.ReadReceiptEvent
			if err := json.Unmarshal(body, &event); err != nil {
				return err
			}
			h.Broadcast(event.Recipients, contracts.EventEnvelope{Type: routingKey, Payload: event})
		case "friend.request":
			var event contracts.FriendRequestEvent
			if err := json.Unmarshal(body, &event); err != nil {
				return err
			}
			h.Broadcast(event.Recipients, contracts.EventEnvelope{Type: routingKey, Payload: event})
		case "sync.notify":
			var event contracts.SyncNotifyEvent
			if err := json.Unmarshal(body, &event); err != nil {
				return err
			}
			h.Broadcast(event.Recipients, contracts.EventEnvelope{Type: routingKey, Payload: event})
		}
		return nil
	}); err != nil {
		log.Fatal(err)
	}
	if err := rabbitClient.Consume("search.events", []string{"search.message.index", "search.conversation.index"}, func(routingKey string, body []byte) error {
		switch routingKey {
		case "search.message.index":
			var event contracts.SearchMessageIndexEvent
			if err := json.Unmarshal(body, &event); err != nil {
				return err
			}
			return chatServices.Search.IndexMessageEvent(context.Background(), event)
		case "search.conversation.index":
			var event contracts.SearchConversationIndexEvent
			if err := json.Unmarshal(body, &event); err != nil {
				return err
			}
			return chatServices.Search.IndexConversationEvent(context.Background(), event)
		}
		return nil
	}); err != nil {
		log.Fatal(err)
	}

	router := gin.New()
	router.Use(httpx.RequestID(), httpx.StructuredLogger(cfg.ServiceName), httpx.Metrics(cfg.ServiceName), gin.Recovery(), httpx.CORS(cfg.CORSOrigins))
	router.GET("/metrics", httpx.MetricsHandler())
	routes.RegisterRealtimeRoutes(router, realtimeController)

	if err := router.Run(":" + cfg.AppPort); err != nil {
		log.Fatal(err)
	}
}

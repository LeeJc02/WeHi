package main

import (
	"context"
	"log"
	"time"

	"github.com/LeeJc02/WeHi/backend/internal/app/chat"
	"github.com/LeeJc02/WeHi/backend/internal/app/repository"
	"github.com/LeeJc02/WeHi/backend/internal/config"
	"github.com/LeeJc02/WeHi/backend/internal/platform/db"
	"github.com/LeeJc02/WeHi/backend/internal/platform/search"
)

func main() {
	cfg := config.Load("reindex", "0")
	gormDB, err := db.OpenGorm(cfg.MySQLDSN, cfg.ServiceName)
	if err != nil {
		log.Fatal(err)
	}
	services := chat.NewServices(repository.New(gormDB), nil, search.New(cfg.ElasticsearchURL), nil, nil, cfg.ElasticsearchMessagesIndex, cfg.ElasticsearchConversationsIndex)
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	if err := services.Search.Reindex(ctx); err != nil {
		log.Fatal(err)
	}
	log.Println("search reindex completed")
}

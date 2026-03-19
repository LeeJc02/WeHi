package main

import (
	"context"
	"log"
	"time"

	"awesomeproject/internal/app/chat"
	"awesomeproject/internal/app/repository"
	"awesomeproject/internal/config"
	"awesomeproject/internal/platform/db"
	"awesomeproject/internal/platform/search"
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

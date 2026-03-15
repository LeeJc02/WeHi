package main

import (
	"log"

	"awesomeproject/backend/internal/config"
	"awesomeproject/backend/internal/database"
	"awesomeproject/backend/internal/routes"
)

func main() {
	cfg := config.Load()

	db, err := database.Open(cfg.DBPath)
	if err != nil {
		log.Fatal(err)
	}

	router := routes.NewRouter(db, cfg)
	if err := router.Run(":" + cfg.AppPort); err != nil {
		log.Fatal(err)
	}
}

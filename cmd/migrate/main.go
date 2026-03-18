package main

import (
	"log"

	"awesomeproject/internal/config"
	"awesomeproject/internal/platform/db"
)

func main() {
	cfg := config.Load("migrate", "0")
	sqlDB, err := db.OpenSQL(cfg.MySQLDSN)
	if err != nil {
		log.Fatal(err)
	}
	defer sqlDB.Close()
	if err := db.RunMigrations(sqlDB, "migrations"); err != nil {
		log.Fatal(err)
	}
	log.Println("migrations applied")
}

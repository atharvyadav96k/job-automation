package main

import (
	"context"
	"log"

	"job-automation/app/internal/config"
	"job-automation/app/internal/db"
	"job-automation/app/internal/migrations"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	ctx := context.Background()
	pool, err := db.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("connect db: %v", err)
	}
	defer pool.Close()

	if err := migrations.Run(ctx, pool); err != nil {
		log.Fatalf("run migrations: %v", err)
	}
	log.Println("migrations up to date")
}

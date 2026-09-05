// One-shot GitHub evidence sync: fetch every non-fork repo for
// GITHUB_USERNAME, upsert it and its derived skill_evidence, then exit.
// Run manually or on a schedule (e.g. daily/weekly cron) to keep the skill
// inventory current — see plan.md Phase 1.5.
package main

import (
	"context"
	"log"

	"job-automation/app/internal/config"
	"job-automation/app/internal/db"
	"job-automation/app/internal/github"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}
	if cfg.GitHubUsername == "" {
		log.Fatal("GITHUB_USERNAME is required")
	}

	ctx := context.Background()
	pool, err := db.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("connect db: %v", err)
	}
	defer pool.Close()

	synced, err := github.Sync(ctx, pool, cfg.GitHubUsername, cfg.GitHubToken)
	if err != nil {
		log.Fatalf("sync: %v", err)
	}
	log.Printf("synced %d repos", synced)
}

// One-shot discovery run: fetch every source once, drain the queue once,
// then exit. Useful for manual testing or an external cron trigger instead
// of running the full server just to scrape.
package main

import (
	"context"
	"flag"
	"log"
	"time"

	"job-automation/app/internal/config"
	"job-automation/app/internal/db"
	"job-automation/app/internal/discovery"
	"job-automation/app/internal/redisqueue"
)

const remotiveResultLimit = 25

func main() {
	drainTimeout := flag.Duration("drain-timeout", 10*time.Second, "how long to wait for the queue to drain")
	flag.Parse()

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

	queue, err := redisqueue.New(cfg.RedisURL)
	if err != nil {
		log.Fatalf("redis queue: %v", err)
	}
	defer queue.Close()

	sources, err := discovery.BuildSources(ctx, pool, cfg.RemotiveEnabled, remotiveResultLimit)
	if err != nil {
		log.Fatalf("build sources: %v", err)
	}
	log.Printf("running %d sources", len(sources))

	fetcher := discovery.NewFetcher(pool, queue, sources)
	queued, err := fetcher.Run(ctx)
	if err != nil {
		log.Fatalf("fetch: %v", err)
	}
	log.Printf("queued %d new jobs", queued)

	drainCtx, cancel := context.WithTimeout(ctx, *drainTimeout)
	defer cancel()
	worker := discovery.NewWorker(pool, queue)
	worker.Run(drainCtx)
	log.Println("drain complete")
}

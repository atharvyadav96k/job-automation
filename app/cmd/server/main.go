package main

import (
	"context"
	"log"
	"net/http"
	"path/filepath"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"job-automation/app/internal/api"
	"job-automation/app/internal/config"
	"job-automation/app/internal/db"
	"job-automation/app/internal/discovery"
	"job-automation/app/internal/llm"
	"job-automation/app/internal/profile"
	"job-automation/app/internal/redisqueue"
	"job-automation/app/internal/tailor"
)

const remotiveResultLimit = 25

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}
	if cfg.BasicAuthUser == "" || cfg.BasicAuthPass == "" {
		log.Fatal("API_BASIC_AUTH_USER and API_BASIC_AUTH_PASS are required")
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
	if err := queue.Ping(ctx); err != nil {
		log.Fatalf("redis unreachable: %v", err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		if err := pool.Ping(r.Context()); err != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			w.Write([]byte("db unreachable"))
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})

	profileHandler := api.NewProfileHandler(profile.NewStore(pool), cfg.ResumeDir)
	profileHandler.Register(mux)

	discoveryHandler := api.NewDiscoveryHandler(pool, queue, cfg.RemotiveEnabled, remotiveResultLimit)
	discoveryHandler.Register(mux)

	geminiClient := llm.NewGeminiClient(cfg.GeminiAPIKey)
	templatePath := filepath.Join(cfg.ResumeDir, "base_resume.docx")
	pipeline := tailor.NewPipeline(pool, geminiClient, cfg.ResumeDir, templatePath)
	tailorHandler := api.NewTailorHandler(pipeline)
	tailorHandler.Register(mux)

	protected := api.CORS(cfg.FrontendOrigin, api.BasicAuth(cfg.BasicAuthUser, cfg.BasicAuthPass, mux))

	worker := discovery.NewWorker(pool, queue)
	go worker.Run(ctx)
	go runScrapeTicker(ctx, pool, queue, cfg)

	log.Printf("listening on %s", cfg.ServerAddr)
	if err := http.ListenAndServe(cfg.ServerAddr, protected); err != nil {
		log.Fatal(err)
	}
}

// runScrapeTicker fetches immediately on startup, then on cfg.ScrapeInterval
// — a goroutine in this process rather than a separate service, since a
// single scheduled fetch is cheap enough not to warrant its own container
// within the 4GB budget.
func runScrapeTicker(ctx context.Context, pool *pgxpool.Pool, queue *redisqueue.Queue, cfg config.Config) {
	runOnce := func() {
		sources, err := discovery.BuildSources(ctx, pool, cfg.RemotiveEnabled, remotiveResultLimit)
		if err != nil {
			log.Printf("scrape ticker: build sources: %v", err)
			return
		}
		fetcher := discovery.NewFetcher(pool, queue, sources)
		queued, err := fetcher.Run(ctx)
		if err != nil {
			log.Printf("scrape ticker: fetch run: %v", err)
			return
		}
		log.Printf("scrape ticker: queued %d new jobs from %d sources", queued, len(sources))
	}

	runOnce()
	ticker := time.NewTicker(cfg.ScrapeInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			runOnce()
		}
	}
}

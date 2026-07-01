package main

import (
	"context"
	"log"
	"net/http"

	"job-automation/app/internal/api"
	"job-automation/app/internal/config"
	"job-automation/app/internal/db"
	"job-automation/app/internal/profile"
)

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

	protected := api.CORS(cfg.FrontendOrigin, api.BasicAuth(cfg.BasicAuthUser, cfg.BasicAuthPass, mux))

	log.Printf("listening on %s", cfg.ServerAddr)
	if err := http.ListenAndServe(cfg.ServerAddr, protected); err != nil {
		log.Fatal(err)
	}
}

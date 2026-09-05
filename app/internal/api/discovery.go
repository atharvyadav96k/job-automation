package api

import (
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"
	"job-automation/app/internal/discovery"
	"job-automation/app/internal/redisqueue"
)

type DiscoveryHandler struct {
	pool       *pgxpool.Pool
	queue      *redisqueue.Queue
	sourcesCfg discovery.SourcesConfig
}

func NewDiscoveryHandler(pool *pgxpool.Pool, queue *redisqueue.Queue, sourcesCfg discovery.SourcesConfig) *DiscoveryHandler {
	return &DiscoveryHandler{pool: pool, queue: queue, sourcesCfg: sourcesCfg}
}

func (h *DiscoveryHandler) Register(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/discovery/run", h.run)
}

// run triggers an immediate fetch on demand — a deliberate click always
// fetches, unlike the scheduled ticker (see runScrapeTicker) which skips
// itself while a review backlog exists. "Scan" is the one place that
// backlog-throttle doesn't apply: it's an explicit ask for more right now.
func (h *DiscoveryHandler) run(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	sources, err := discovery.BuildSources(ctx, h.pool, h.sourcesCfg)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	fetcher := discovery.NewFetcher(h.pool, h.queue, sources)
	queued, err := fetcher.Run(ctx)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"sources": len(sources), "queued": queued})
}

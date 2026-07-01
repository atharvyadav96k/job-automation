package api

import (
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"
	"job-automation/app/internal/discovery"
	"job-automation/app/internal/redisqueue"
)

type DiscoveryHandler struct {
	pool            *pgxpool.Pool
	queue           *redisqueue.Queue
	remotiveEnabled bool
	remotiveLimit   int
}

func NewDiscoveryHandler(pool *pgxpool.Pool, queue *redisqueue.Queue, remotiveEnabled bool, remotiveLimit int) *DiscoveryHandler {
	return &DiscoveryHandler{pool: pool, queue: queue, remotiveEnabled: remotiveEnabled, remotiveLimit: remotiveLimit}
}

func (h *DiscoveryHandler) Register(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/discovery/run", h.run)
}

// run triggers an immediate fetch on demand, instead of waiting for the
// next scheduled tick — useful right after adding a new company, and
// cheaper than lowering the global interval just to test one source.
func (h *DiscoveryHandler) run(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	sources, err := discovery.BuildSources(ctx, h.pool, h.remotiveEnabled, h.remotiveLimit)
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

package discovery

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/jackc/pgx/v5/pgxpool"
	"job-automation/app/internal/redisqueue"
)

type Fetcher struct {
	pool    *pgxpool.Pool
	queue   *redisqueue.Queue
	sources []Source
}

func NewFetcher(pool *pgxpool.Pool, queue *redisqueue.Queue, sources []Source) *Fetcher {
	return &Fetcher{pool: pool, queue: queue, sources: sources}
}

// Run fetches every source, drops jobs already present in the jobs table
// (dedup by URL, the cheap check before queueing), and pushes the rest onto
// the Redis queue for the worker to clean and persist.
func (f *Fetcher) Run(ctx context.Context) (queued int, err error) {
	for _, src := range f.sources {
		raw, err := src.Fetch(ctx)
		if err != nil {
			log.Printf("discovery: source %s failed: %v", src.Name(), err)
			continue
		}
		skipped := 0
		for _, job := range raw {
			if job.URL == "" {
				continue
			}
			if IsSeniorLevelTitle(job.Title) {
				skipped++
				continue
			}
			var exists bool
			if err := f.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM jobs WHERE url = $1)`, job.URL).Scan(&exists); err != nil {
				return queued, fmt.Errorf("check existing job %s: %w", job.URL, err)
			}
			if exists {
				continue
			}
			payload, err := json.Marshal(job)
			if err != nil {
				return queued, fmt.Errorf("marshal job %s: %w", job.URL, err)
			}
			if err := f.queue.Push(ctx, redisqueue.DiscoveredJobsKey, string(payload)); err != nil {
				return queued, fmt.Errorf("enqueue job %s: %w", job.URL, err)
			}
			queued++
		}
		log.Printf("discovery: source %s fetched %d jobs (%d skipped as too senior)", src.Name(), len(raw), skipped)
	}
	return queued, nil
}

// AvailableJobsCount returns how many jobs still need action — not yet
// applied to and not skipped. Scraping is throttled on this: no point
// piling up more discovered jobs while there's already a backlog waiting
// for review/tailoring/application.
func AvailableJobsCount(ctx context.Context, pool *pgxpool.Pool) (int, error) {
	var count int
	err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM jobs WHERE status NOT IN ('applied', 'skipped')`).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count available jobs: %w", err)
	}
	return count, nil
}

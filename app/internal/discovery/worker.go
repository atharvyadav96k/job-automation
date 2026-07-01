package discovery

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/jackc/pgx/v5/pgxpool"
	"job-automation/app/internal/redisqueue"
)

type Worker struct {
	pool  *pgxpool.Pool
	queue *redisqueue.Queue
}

func NewWorker(pool *pgxpool.Pool, queue *redisqueue.Queue) *Worker {
	return &Worker{pool: pool, queue: queue}
}

// Run blocks, consuming the discovery queue until ctx is cancelled.
func (w *Worker) Run(ctx context.Context) {
	for {
		payload, err := w.queue.Pop(ctx, redisqueue.DiscoveredJobsKey)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			log.Printf("discovery worker: pop failed: %v", err)
			continue
		}
		if err := w.process(ctx, payload); err != nil {
			log.Printf("discovery worker: process failed: %v", err)
		}
	}
}

func (w *Worker) process(ctx context.Context, payload string) error {
	var job RawJob
	if err := json.Unmarshal([]byte(payload), &job); err != nil {
		return fmt.Errorf("unmarshal queued job: %w", err)
	}
	if job.CompanyName == "" {
		job.CompanyName = "Unknown"
	}

	companyID, err := w.getOrCreateCompany(ctx, job.CompanyName, job.ATSPlatform)
	if err != nil {
		return fmt.Errorf("get or create company %s: %w", job.CompanyName, err)
	}

	clean := CleanHTML(job.DescriptionHTML)

	_, err = w.pool.Exec(ctx, `
		INSERT INTO jobs (company_id, title, url, description_raw, description_clean, source, status)
		VALUES ($1, $2, $3, $4, $5, $6, 'new')
		ON CONFLICT (url) DO NOTHING
	`, companyID, job.Title, job.URL, job.DescriptionHTML, clean, job.Source)
	if err != nil {
		return fmt.Errorf("insert job %s: %w", job.URL, err)
	}
	log.Printf("discovery worker: stored job %q (%s)", job.Title, job.URL)
	return nil
}

func (w *Worker) getOrCreateCompany(ctx context.Context, name, atsPlatform string) (int64, error) {
	var id int64
	err := w.pool.QueryRow(ctx, `
		INSERT INTO companies (name, ats_platform)
		VALUES ($1, $2)
		ON CONFLICT (name) DO UPDATE SET name = EXCLUDED.name
		RETURNING id
	`, name, atsPlatform).Scan(&id)
	return id, err
}

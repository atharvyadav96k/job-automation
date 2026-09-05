package github

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Sync fetches every non-fork repo for the configured GitHub user, upserts
// it into repos, and derives skill_evidence from its language + topics.
// Forks are skipped — they're not evidence of the account owner's own work.
// Returns the number of repos synced.
func Sync(ctx context.Context, pool *pgxpool.Pool, username, token string) (int, error) {
	client := NewClient(username, token)
	repos, err := client.ListRepos()
	if err != nil {
		return 0, fmt.Errorf("list repos: %w", err)
	}

	synced := 0
	for _, r := range repos {
		if r.Fork {
			continue
		}

		var lastCommit *time.Time
		if r.PushedAt != "" {
			if t, err := time.Parse(time.RFC3339, r.PushedAt); err == nil {
				lastCommit = &t
			}
		}

		topicsJSON, err := json.Marshal(r.Topics)
		if err != nil {
			return synced, fmt.Errorf("marshal topics for repo %s: %w", r.Name, err)
		}

		var repoID int64
		err = pool.QueryRow(ctx, `
			INSERT INTO repos (github_id, name, github_url, description, language, topics, stars, last_commit_at, synced_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, now())
			ON CONFLICT (github_id) DO UPDATE SET
				name = EXCLUDED.name,
				github_url = EXCLUDED.github_url,
				description = EXCLUDED.description,
				language = EXCLUDED.language,
				topics = EXCLUDED.topics,
				stars = EXCLUDED.stars,
				last_commit_at = EXCLUDED.last_commit_at,
				synced_at = now()
			RETURNING id
		`, r.ID, r.Name, r.HTMLURL, r.Description, r.Language, topicsJSON, r.Stars, lastCommit).Scan(&repoID)
		if err != nil {
			return synced, fmt.Errorf("upsert repo %s: %w", r.Name, err)
		}

		skills := skillNamesForRepo(r)
		for _, skill := range skills {
			if _, err := pool.Exec(ctx, `
				INSERT INTO skill_evidence (skill_name, repo_id, confidence)
				VALUES ($1, $2, 'inferred')
				ON CONFLICT (skill_name, repo_id) DO NOTHING
			`, skill, repoID); err != nil {
				return synced, fmt.Errorf("insert skill_evidence %s for repo %s: %w", skill, r.Name, err)
			}
		}

		synced++
	}
	return synced, nil
}

// skillNamesForRepo derives skill keywords from a repo's primary language and
// topics — simple keyword extraction, matching the plan's "start simple"
// guidance rather than parsing dependency manifests.
func skillNamesForRepo(r Repo) []string {
	seen := make(map[string]bool)
	var skills []string
	add := func(name string) {
		if name == "" || seen[name] {
			return
		}
		seen[name] = true
		skills = append(skills, name)
	}

	add(r.Language)
	for _, t := range r.Topics {
		add(t)
	}
	return skills
}

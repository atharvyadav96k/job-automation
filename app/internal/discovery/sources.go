package discovery

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
	"job-automation/app/internal/llm"
)

// LoadCompanySources builds a Source per row in companies whose
// ats_platform is greenhouse or lever. careers_url holds the board slug for
// those platforms (not a full URL) — set it when adding a company to track.
func LoadCompanySources(ctx context.Context, pool *pgxpool.Pool) ([]Source, error) {
	rows, err := pool.Query(ctx, `
		SELECT name, careers_url, ats_platform FROM companies
		WHERE ats_platform IN ('greenhouse', 'lever') AND careers_url IS NOT NULL AND careers_url != ''
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sources []Source
	for rows.Next() {
		var name, slug, platform string
		if err := rows.Scan(&name, &slug, &platform); err != nil {
			return nil, err
		}
		switch platform {
		case "greenhouse":
			sources = append(sources, NewGreenhouseSource(name, slug))
		case "lever":
			sources = append(sources, NewLeverSource(name, slug))
		}
	}
	return sources, rows.Err()
}

// SourcesConfig controls which discovery sources BuildSources assembles.
type SourcesConfig struct {
	RemotiveEnabled bool
	RemotiveLimit   int
	AISearchEnabled bool
	AISearchLimit   int
	AIClient        llm.GroundedSearcher // nil if the configured LLM provider doesn't support grounding
}

// BuildSources assembles every configured source fresh: company rows can
// change at any time, and the profile's skills (which drive the Remotive
// and AI-search queries) can too, so this is called once per fetch run
// rather than cached.
func BuildSources(ctx context.Context, pool *pgxpool.Pool, cfg SourcesConfig) ([]Source, error) {
	sources, err := LoadCompanySources(ctx, pool)
	if err != nil {
		return nil, err
	}

	var skillsJSON []byte
	haveSkills := pool.QueryRow(ctx, `SELECT skills FROM user_profile WHERE id = 1`).Scan(&skillsJSON) == nil

	if cfg.RemotiveEnabled && haveSkills {
		query, err := RemotiveQueryFromProfile(skillsJSON, 3)
		if err == nil && query != "" {
			sources = append(sources, NewRemotiveSource(query, cfg.RemotiveLimit))
		}
	}

	if cfg.AISearchEnabled && cfg.AIClient != nil && haveSkills {
		desc, err := AllSkillNames(skillsJSON)
		if err == nil && desc != "" {
			sources = append(sources, NewAISearchSource(cfg.AIClient, desc, cfg.AISearchLimit))
		}
	}
	return sources, nil
}

// AllSkillNames joins every skill name in the profile (unlike
// RemotiveQueryFromProfile's top-N, the AI search prompt benefits from the
// full picture rather than a short keyword query).
func AllSkillNames(skillsJSON []byte) (string, error) {
	var skills []struct {
		Name string `json:"name"`
	}
	if err := json.Unmarshal(skillsJSON, &skills); err != nil {
		return "", err
	}
	names := make([]string, 0, len(skills))
	for _, s := range skills {
		if s.Name != "" {
			names = append(names, s.Name)
		}
	}
	return strings.Join(names, ", "), nil
}

// RemotiveQueryFromProfile builds a search query from the profile's top
// skills, so the job-board source has something useful to search for
// without any company being manually configured.
func RemotiveQueryFromProfile(skillsJSON []byte, maxSkills int) (string, error) {
	var skills []struct {
		Name string `json:"name"`
	}
	if err := json.Unmarshal(skillsJSON, &skills); err != nil {
		return "", err
	}
	names := make([]string, 0, maxSkills)
	for i, s := range skills {
		if i >= maxSkills {
			break
		}
		if s.Name != "" {
			names = append(names, s.Name)
		}
	}
	return strings.Join(names, " "), nil
}

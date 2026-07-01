package discovery

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
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

// BuildSources assembles every configured source fresh: company rows can
// change at any time, and the profile's skills (which drive the Remotive
// query) can too, so this is called once per fetch run rather than cached.
func BuildSources(ctx context.Context, pool *pgxpool.Pool, remotiveEnabled bool, remotiveLimit int) ([]Source, error) {
	sources, err := LoadCompanySources(ctx, pool)
	if err != nil {
		return nil, err
	}

	if remotiveEnabled {
		var skillsJSON []byte
		err := pool.QueryRow(ctx, `SELECT skills FROM user_profile WHERE id = 1`).Scan(&skillsJSON)
		if err == nil {
			query, err := RemotiveQueryFromProfile(skillsJSON, 3)
			if err == nil && query != "" {
				sources = append(sources, NewRemotiveSource(query, remotiveLimit))
			}
		}
	}
	return sources, nil
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

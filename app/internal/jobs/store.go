package jobs

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Store struct {
	pool *pgxpool.Pool
}

func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

type Summary struct {
	ID               int64     `json:"id"`
	Title            string    `json:"title"`
	CompanyName      string    `json:"company_name"`
	Status           string    `json:"status"`
	URL              string    `json:"url"`
	DiscoveredAt     time.Time `json:"discovered_at"`
	VersionCount     int       `json:"version_count"`
	LatestMatchScore *int      `json:"latest_match_score"`
	LatestATSScore   *int      `json:"latest_ats_score"`
	Approved         bool      `json:"approved"`
}

// List returns every job with a summary of its latest tailoring attempt (if
// any), most recently discovered first.
func (s *Store) List(ctx context.Context) ([]Summary, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT
			j.id, j.title, c.name, j.status, j.url, j.discovered_at,
			COUNT(rv.id) AS version_count,
			(SELECT rv2.match_score FROM resume_versions rv2 WHERE rv2.job_id = j.id ORDER BY rv2.version_number DESC LIMIT 1),
			(SELECT rv2.ats_score FROM resume_versions rv2 WHERE rv2.job_id = j.id ORDER BY rv2.version_number DESC LIMIT 1),
			COALESCE(bool_or(rv.approved), false)
		FROM jobs j
		JOIN companies c ON c.id = j.company_id
		LEFT JOIN resume_versions rv ON rv.job_id = j.id
		GROUP BY j.id, c.name
		ORDER BY j.discovered_at DESC
		LIMIT 200
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Summary
	for rows.Next() {
		var s Summary
		if err := rows.Scan(&s.ID, &s.Title, &s.CompanyName, &s.Status, &s.URL, &s.DiscoveredAt,
			&s.VersionCount, &s.LatestMatchScore, &s.LatestATSScore, &s.Approved); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

type Detail struct {
	ID               int64           `json:"id"`
	Title            string          `json:"title"`
	CompanyName      string          `json:"company_name"`
	Status           string          `json:"status"`
	URL              string          `json:"url"`
	DescriptionClean string          `json:"description_clean"`
	DiscoveredAt     time.Time       `json:"discovered_at"`
	JobContext       *JobContext     `json:"job_context"`
	ResumeVersions   []ResumeVersion `json:"resume_versions"`
}

type JobContext struct {
	ID              int64    `json:"id"`
	CompanySummary  string   `json:"company_summary"`
	KeyRequirements []string `json:"key_requirements"`
	InferredTone    string   `json:"inferred_tone"`
}

type ATSBreakdown struct {
	MatchedKeywords []string `json:"matched_keywords"`
	MissingKeywords []string `json:"missing_keywords"`
	FormattingNotes []string `json:"formatting_notes"`
}

type TailoredContent struct {
	Skills      map[string]string `json:"skills"`
	Exp1Bullets []string          `json:"exp1_bullets"`
	Projects    []struct {
		Title string `json:"title"`
		Tech  string `json:"tech"`
		Link  string `json:"link"`
	} `json:"projects"`
}

type ResumeVersion struct {
	ID              int64           `json:"id"`
	VersionNumber   int             `json:"version_number"`
	MatchScore      *int            `json:"match_score"`
	ATSScore        *int            `json:"ats_score"`
	ATSBreakdown    ATSBreakdown    `json:"ats_score_breakdown"`
	TailoredContent TailoredContent `json:"tailored_content"`
	CoverLetter     string          `json:"generated_cover_letter_text"`
	ChangesSummary  string          `json:"changes_summary"`
	Reasoning       string          `json:"reasoning"`
	ModelUsed       string          `json:"model_used"`
	Approved        bool            `json:"approved"`
	IsActive        bool            `json:"is_active"`
	CreatedAt       time.Time       `json:"created_at"`
}

func (s *Store) Get(ctx context.Context, id int64) (Detail, error) {
	var d Detail
	err := s.pool.QueryRow(ctx, `
		SELECT j.id, j.title, c.name, j.status, j.url, j.description_clean, j.discovered_at
		FROM jobs j JOIN companies c ON c.id = j.company_id
		WHERE j.id = $1
	`, id).Scan(&d.ID, &d.Title, &d.CompanyName, &d.Status, &d.URL, &d.DescriptionClean, &d.DiscoveredAt)
	if err != nil {
		return Detail{}, fmt.Errorf("load job: %w", err)
	}

	var jc JobContext
	var keyReqJSON []byte
	err = s.pool.QueryRow(ctx, `
		SELECT id, company_summary, key_requirements, inferred_tone
		FROM job_context WHERE job_id = $1 ORDER BY created_at DESC LIMIT 1
	`, id).Scan(&jc.ID, &jc.CompanySummary, &keyReqJSON, &jc.InferredTone)
	if err == nil {
		if err := json.Unmarshal(keyReqJSON, &jc.KeyRequirements); err != nil {
			return Detail{}, fmt.Errorf("unmarshal key_requirements: %w", err)
		}
		d.JobContext = &jc
	} else if err != pgx.ErrNoRows {
		return Detail{}, fmt.Errorf("load job_context: %w", err)
	}

	rows, err := s.pool.Query(ctx, `
		SELECT id, version_number, match_score, ats_score, ats_score_breakdown, tailored_content,
			generated_cover_letter_text, changes_summary, reasoning, model_used, approved, is_active, created_at
		FROM resume_versions WHERE job_id = $1 ORDER BY version_number ASC
	`, id)
	if err != nil {
		return Detail{}, fmt.Errorf("load resume_versions: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var rv ResumeVersion
		var breakdownJSON, contentJSON []byte
		if err := rows.Scan(&rv.ID, &rv.VersionNumber, &rv.MatchScore, &rv.ATSScore, &breakdownJSON, &contentJSON,
			&rv.CoverLetter, &rv.ChangesSummary, &rv.Reasoning, &rv.ModelUsed, &rv.Approved, &rv.IsActive, &rv.CreatedAt); err != nil {
			return Detail{}, fmt.Errorf("scan resume_version: %w", err)
		}
		if len(breakdownJSON) > 0 {
			if err := json.Unmarshal(breakdownJSON, &rv.ATSBreakdown); err != nil {
				return Detail{}, fmt.Errorf("unmarshal ats_score_breakdown: %w", err)
			}
		}
		if len(contentJSON) > 0 {
			if err := json.Unmarshal(contentJSON, &rv.TailoredContent); err != nil {
				return Detail{}, fmt.Errorf("unmarshal tailored_content: %w", err)
			}
		}
		d.ResumeVersions = append(d.ResumeVersions, rv)
	}
	return d, rows.Err()
}

// Approve marks the given resume_version as the chosen one for its job:
// approved=true and is_active=true on it, is_active=false on every sibling
// version — the plan's "approving sets is_active = true on the chosen
// version" rule, which may promote an older version over the latest one.
func (s *Store) Approve(ctx context.Context, jobID, versionID int64) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, `UPDATE resume_versions SET is_active = false WHERE job_id = $1`, jobID); err != nil {
		return err
	}
	tag, err := tx.Exec(ctx, `
		UPDATE resume_versions SET approved = true, approved_at = now(), is_active = true
		WHERE id = $1 AND job_id = $2
	`, versionID, jobID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("resume_version %d not found for job %d", versionID, jobID)
	}
	return tx.Commit(ctx)
}

// Reject marks the job as skipped — none of its resume_versions get applied.
func (s *Store) Reject(ctx context.Context, jobID int64) error {
	_, err := s.pool.Exec(ctx, `UPDATE jobs SET status = 'skipped' WHERE id = $1`, jobID)
	return err
}

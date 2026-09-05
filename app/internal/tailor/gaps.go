package tailor

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

// DetectSkillGaps diffs a job's key_requirements (already extracted into
// job_context) against the GitHub-derived skill_evidence inventory, and
// records any unmatched requirement as a skill_gap for manual review —
// never auto-filled with fabricated evidence, per plan.md.
func DetectSkillGaps(ctx context.Context, pool *pgxpool.Pool, jobID int64, keyRequirements []string) error {
	rows, err := pool.Query(ctx, `SELECT DISTINCT skill_name FROM skill_evidence`)
	if err != nil {
		return fmt.Errorf("load skill evidence: %w", err)
	}
	var known []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			rows.Close()
			return fmt.Errorf("scan skill evidence: %w", err)
		}
		known = append(known, strings.ToLower(name))
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate skill evidence: %w", err)
	}

	for _, req := range keyRequirements {
		if hasEvidence(req, known) {
			continue
		}
		if _, err := pool.Exec(ctx, `
			INSERT INTO skill_gaps (job_id, missing_skill)
			VALUES ($1, $2)
			ON CONFLICT (job_id, missing_skill) DO NOTHING
		`, jobID, req); err != nil {
			return fmt.Errorf("insert skill_gap %q: %w", req, err)
		}
	}
	return nil
}

type SkillGap struct {
	ID           int64  `json:"id"`
	MissingSkill string `json:"missing_skill"`
	Reviewed     bool   `json:"reviewed"`
}

// ListSkillGaps returns the skill gaps recorded for a job, most recent first.
func ListSkillGaps(ctx context.Context, pool *pgxpool.Pool, jobID int64) ([]SkillGap, error) {
	rows, err := pool.Query(ctx, `
		SELECT id, missing_skill, reviewed FROM skill_gaps
		WHERE job_id = $1 ORDER BY created_at DESC
	`, jobID)
	if err != nil {
		return nil, fmt.Errorf("list skill gaps: %w", err)
	}
	defer rows.Close()

	var gaps []SkillGap
	for rows.Next() {
		var g SkillGap
		if err := rows.Scan(&g.ID, &g.MissingSkill, &g.Reviewed); err != nil {
			return nil, fmt.Errorf("scan skill gap: %w", err)
		}
		gaps = append(gaps, g)
	}
	return gaps, rows.Err()
}

// MarkGapReviewed records that the account owner has manually looked at a
// skill gap and decided how to handle it (e.g. build something real, or
// accept the gap) — the gap itself is never auto-resolved.
func MarkGapReviewed(ctx context.Context, pool *pgxpool.Pool, gapID int64) error {
	_, err := pool.Exec(ctx, `UPDATE skill_gaps SET reviewed = true WHERE id = $1`, gapID)
	if err != nil {
		return fmt.Errorf("mark skill gap reviewed: %w", err)
	}
	return nil
}

// hasEvidence does a simple case-insensitive substring match in either
// direction — good enough for short keyword-style requirements/skill names,
// matching the plan's "simple keyword extraction to start" guidance.
func hasEvidence(requirement string, known []string) bool {
	req := strings.ToLower(requirement)
	for _, skill := range known {
		if skill == "" {
			continue
		}
		if strings.Contains(req, skill) || strings.Contains(skill, req) {
			return true
		}
	}
	return false
}

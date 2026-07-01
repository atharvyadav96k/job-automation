package tailor

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/jackc/pgx/v5/pgxpool"
	"job-automation/app/internal/llm"
	"job-automation/app/internal/resume"
)

type Pipeline struct {
	pool         *pgxpool.Pool
	client       llm.Client
	resumeDir    string
	templatePath string
}

func NewPipeline(pool *pgxpool.Pool, client llm.Client, resumeDir, templatePath string) *Pipeline {
	return &Pipeline{pool: pool, client: client, resumeDir: resumeDir, templatePath: templatePath}
}

type Result struct {
	ResumeVersionID int64
	VersionNumber   int
	MatchScore      int
	ATSScore        int
	GeneratedPath   string
}

// Run tailors one job: extract-or-reuse job_context, call the LLM, fill the
// docx template, and store a new resume_versions row (approved=false).
func (p *Pipeline) Run(ctx context.Context, jobID int64) (Result, error) {
	var jobTitle, companyName, descriptionClean string
	err := p.pool.QueryRow(ctx, `
		SELECT j.title, c.name, j.description_clean
		FROM jobs j JOIN companies c ON c.id = j.company_id
		WHERE j.id = $1
	`, jobID).Scan(&jobTitle, &companyName, &descriptionClean)
	if err != nil {
		return Result{}, fmt.Errorf("load job %d: %w", jobID, err)
	}

	jobCtxID, jobCtx, err := p.getOrCreateJobContext(ctx, jobID, jobTitle, companyName, descriptionClean)
	if err != nil {
		return Result{}, fmt.Errorf("job context: %w", err)
	}

	var skillsJSON []byte
	var experienceJSON []byte
	err = p.pool.QueryRow(ctx, `SELECT skills, experience FROM user_profile WHERE id = 1`).Scan(&skillsJSON, &experienceJSON)
	if err != nil {
		return Result{}, fmt.Errorf("load profile: %w", err)
	}

	exp1Bullets, err := firstExperienceBullets(experienceJSON)
	if err != nil {
		return Result{}, fmt.Errorf("read experience bullets: %w", err)
	}

	tailored, usage, err := Tailor(ctx, p.client, jobTitle, companyName, descriptionClean, jobCtx, skillsJSON, exp1Bullets)
	if err != nil {
		return Result{}, err
	}

	docxBytes, err := p.fillTemplate(tailored)
	if err != nil {
		return Result{}, fmt.Errorf("fill docx template: %w", err)
	}

	version, err := p.nextVersionNumber(ctx, jobID)
	if err != nil {
		return Result{}, fmt.Errorf("determine version number: %w", err)
	}

	generatedDir := filepath.Join(p.resumeDir, "generated")
	if err := os.MkdirAll(generatedDir, 0o755); err != nil {
		return Result{}, fmt.Errorf("create generated dir: %w", err)
	}
	generatedPath := filepath.Join(generatedDir, fmt.Sprintf("job_%d_v%d.docx", jobID, version))
	if err := os.WriteFile(generatedPath, docxBytes, 0o644); err != nil {
		return Result{}, fmt.Errorf("write generated docx: %w", err)
	}

	breakdownJSON, err := json.Marshal(tailored.ATSBreakdown)
	if err != nil {
		return Result{}, fmt.Errorf("marshal ats breakdown: %w", err)
	}

	tx, err := p.pool.Begin(ctx)
	if err != nil {
		return Result{}, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, `UPDATE resume_versions SET is_active = false WHERE job_id = $1`, jobID); err != nil {
		return Result{}, fmt.Errorf("deactivate prior versions: %w", err)
	}

	var resumeVersionID int64
	err = tx.QueryRow(ctx, `
		INSERT INTO resume_versions (
			job_id, job_context_id, version_number,
			generated_resume_docx_path, generated_cover_letter_text,
			ats_score, ats_score_breakdown, changes_summary, reasoning,
			model_used, prompt_tokens, completion_tokens,
			approved, is_active
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, false, true)
		RETURNING id
	`, jobID, jobCtxID, version, generatedPath, tailored.CoverLetter,
		tailored.ATSScore, breakdownJSON, tailored.ChangesSummary, tailored.Reasoning,
		llmModelName(p.client), usage.PromptTokens, usage.CompletionTokens,
	).Scan(&resumeVersionID)
	if err != nil {
		return Result{}, fmt.Errorf("insert resume_version: %w", err)
	}

	if _, err := tx.Exec(ctx, `UPDATE jobs SET status = 'scored' WHERE id = $1`, jobID); err != nil {
		return Result{}, fmt.Errorf("update job status: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return Result{}, fmt.Errorf("commit tx: %w", err)
	}

	return Result{
		ResumeVersionID: resumeVersionID,
		VersionNumber:   version,
		MatchScore:      tailored.MatchScore,
		ATSScore:        tailored.ATSScore,
		GeneratedPath:   generatedPath,
	}, nil
}

func (p *Pipeline) getOrCreateJobContext(ctx context.Context, jobID int64, jobTitle, companyName, descriptionClean string) (int64, JobContextResult, error) {
	var id int64
	var jc JobContextResult
	var keyReqJSON []byte
	err := p.pool.QueryRow(ctx, `
		SELECT id, company_summary, key_requirements, inferred_tone
		FROM job_context WHERE job_id = $1 ORDER BY created_at DESC LIMIT 1
	`, jobID).Scan(&id, &jc.CompanySummary, &keyReqJSON, &jc.InferredTone)
	if err == nil {
		if err := json.Unmarshal(keyReqJSON, &jc.KeyRequirements); err != nil {
			return 0, JobContextResult{}, fmt.Errorf("unmarshal cached key_requirements: %w", err)
		}
		return id, jc, nil
	}

	extracted, _, err := ExtractJobContext(ctx, p.client, jobTitle, companyName, descriptionClean)
	if err != nil {
		return 0, JobContextResult{}, err
	}
	reqJSON, err := json.Marshal(extracted.KeyRequirements)
	if err != nil {
		return 0, JobContextResult{}, fmt.Errorf("marshal key_requirements: %w", err)
	}

	err = p.pool.QueryRow(ctx, `
		INSERT INTO job_context (job_id, company_summary, key_requirements, inferred_tone, model_used)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id
	`, jobID, extracted.CompanySummary, reqJSON, extracted.InferredTone, llmModelName(p.client)).Scan(&id)
	if err != nil {
		return 0, JobContextResult{}, fmt.Errorf("insert job_context: %w", err)
	}
	return id, extracted, nil
}

func (p *Pipeline) nextVersionNumber(ctx context.Context, jobID int64) (int, error) {
	var max int
	err := p.pool.QueryRow(ctx, `SELECT COALESCE(MAX(version_number), 0) FROM resume_versions WHERE job_id = $1`, jobID).Scan(&max)
	if err != nil {
		return 0, err
	}
	return max + 1, nil
}

func (p *Pipeline) fillTemplate(t TailorResult) ([]byte, error) {
	values := map[string]string{}
	for category, key := range resume.SkillCategoryKeys {
		values[key] = t.Skills[category]
	}
	for i := 0; i < resume.Exp1BulletCount; i++ {
		bullet := ""
		if i < len(t.Exp1Bullets) {
			bullet = t.Exp1Bullets[i]
		}
		values[resume.Exp1Bullet(i+1)] = bullet
	}
	return resume.Fill(p.templatePath, values)
}

func firstExperienceBullets(experienceJSON []byte) ([]string, error) {
	var experience []struct {
		Bullets []string `json:"bullets"`
	}
	if err := json.Unmarshal(experienceJSON, &experience); err != nil {
		return nil, err
	}
	if len(experience) == 0 {
		return nil, fmt.Errorf("profile has no experience entries")
	}
	return experience[0].Bullets, nil
}

func llmModelName(client llm.Client) string {
	type named interface{ Model() string }
	if n, ok := client.(named); ok {
		return n.Model()
	}
	return "unknown"
}

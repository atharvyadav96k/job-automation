package tailor

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"job-automation/app/internal/llm"
)

const tailorPrompt = `You are tailoring a resume and writing a cover letter for a specific job. You must
only use the candidate's real skills and real experience bullets provided below — never
invent skills, employers, or achievements they don't have. You may rephrase, reorder, or
select a relevant subset of the real bullets to emphasize what matters for this job.

Job title: %s
Company: %s
Company context: %s
Inferred tone for cover letter: %s
Key requirements from the posting: %s
Job description:
%s

Candidate's real skills by category (JSON, name -> category):
%s

Candidate's real experience bullets for their current/most recent role (choose and
rephrase up to %d of these, do not invent new ones):
%s

Return ONLY JSON matching this shape, no markdown fences:
{
  "match_score": <0-100 integer, your honest estimate of how well this candidate fits>,
  "skills": {"backend": "comma-separated skills for this category, or empty string", "frontend": "...", "database": "...", "integration": "...", "devops": "...", "cloud": "...", "tools": "..."},
  "exp1_bullets": ["up to %d tailored bullet strings, most relevant first, only from the real bullets provided"],
  "cover_letter": "full cover letter text, matching the inferred tone",
  "ats_score": <0-100 integer, self-estimated ATS keyword-match score>,
  "ats_breakdown": {
    "matched_keywords": ["keywords from the job description found in the tailored resume"],
    "missing_keywords": ["important keywords from the job description NOT covered by the resume"],
    "formatting_notes": ["any ATS formatting concerns, or empty array if none"]
  },
  "changes_summary": "1-2 sentences: what changed vs. the candidate's base resume",
  "reasoning": "1-3 sentences: why these bullets/skills were chosen for this job specifically"
}`

type skillInput struct {
	Name     string `json:"name"`
	Category string `json:"category"`
}

// Tailor calls Gemini once to produce everything Phase 2 needs: docx
// content, cover letter, and ATS score breakdown, matching the plan's
// single-prompt design.
func Tailor(
	ctx context.Context,
	client llm.Client,
	jobTitle, companyName, descriptionClean string,
	jobCtx JobContextResult,
	skillsJSON []byte,
	exp1Bullets []string,
) (TailorResult, llm.Usage, error) {
	var skills []skillInput
	if err := json.Unmarshal(skillsJSON, &skills); err != nil {
		return TailorResult{}, llm.Usage{}, fmt.Errorf("unmarshal profile skills: %w", err)
	}
	skillsPretty, err := json.MarshalIndent(skills, "", "  ")
	if err != nil {
		return TailorResult{}, llm.Usage{}, fmt.Errorf("marshal skills for prompt: %w", err)
	}

	bulletsList := "- " + strings.Join(exp1Bullets, "\n- ")
	requirementsList := strings.Join(jobCtx.KeyRequirements, ", ")

	prompt := fmt.Sprintf(tailorPrompt,
		jobTitle, companyName, jobCtx.CompanySummary, jobCtx.InferredTone, requirementsList, descriptionClean,
		skillsPretty, len(exp1Bullets), bulletsList, len(exp1Bullets),
	)

	var result TailorResult
	usage, err := client.GenerateJSON(ctx, prompt, &result)
	if err != nil {
		return TailorResult{}, usage, fmt.Errorf("tailor resume: %w", err)
	}
	return result, usage, nil
}

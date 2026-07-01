package tailor

import (
	"context"
	"fmt"

	"job-automation/app/internal/llm"
)

const jobContextPrompt = `You are analyzing a job posting to extract structured context for a resume-tailoring pipeline.

Job title: %s
Company: %s
Job description:
%s

Return ONLY JSON matching this shape, no markdown fences:
{
  "company_summary": "2-3 sentences: industry, size/stage signals, culture signals inferred from the posting",
  "key_requirements": ["short phrase for each must-have requirement or key skill, most important first"],
  "inferred_tone": "one or two words, e.g. 'formal', 'startup-casual', 'corporate'"
}`

// ExtractJobContext runs once per job (cacheable) and is reused by every
// resume_versions attempt for that job — see .claude/skills/ats-scoring.
func ExtractJobContext(ctx context.Context, client llm.Client, jobTitle, companyName, descriptionClean string) (JobContextResult, llm.Usage, error) {
	prompt := fmt.Sprintf(jobContextPrompt, jobTitle, companyName, descriptionClean)
	var result JobContextResult
	usage, err := client.GenerateJSON(ctx, prompt, &result)
	if err != nil {
		return JobContextResult{}, usage, fmt.Errorf("extract job context: %w", err)
	}
	return result, usage, nil
}

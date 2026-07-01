package tailor

// JobContextResult is the cacheable, per-job extraction — computed once and
// reused across re-tailoring attempts (see .claude/skills/ats-scoring).
type JobContextResult struct {
	CompanySummary  string   `json:"company_summary"`
	KeyRequirements []string `json:"key_requirements"`
	InferredTone    string   `json:"inferred_tone"`
}

// TailorResult is the structured output of one tailoring attempt: content
// for the docx template, the cover letter, and the self-estimated ATS
// breakdown, all in one call per the build plan.
type TailorResult struct {
	MatchScore     int               `json:"match_score"`
	Skills         map[string]string `json:"skills"` // keyed by category: backend, frontend, database, integration, devops, cloud, tools
	Exp1Bullets    []string          `json:"exp1_bullets"`
	CoverLetter    string            `json:"cover_letter"`
	ATSScore       int               `json:"ats_score"`
	ATSBreakdown   ATSBreakdown      `json:"ats_breakdown"`
	ChangesSummary string            `json:"changes_summary"`
	Reasoning      string            `json:"reasoning"`
}

type ATSBreakdown struct {
	MatchedKeywords []string `json:"matched_keywords"`
	MissingKeywords []string `json:"missing_keywords"`
	FormattingNotes []string `json:"formatting_notes"`
}

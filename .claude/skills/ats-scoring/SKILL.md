---
name: ats-scoring
description: Use this skill for any code that computes, displays, or explains the ATS (Applicant Tracking System) match score for a tailored resume. Use when implementing the scoring logic itself, when adding the score breakdown to the review UI, or when the user wants to tune scoring accuracy.
---

# ATS Match Scoring

## What "ATS score" means here
This is a **self-estimated** heuristic, not a connection to any real ATS vendor's actual
scoring engine (those are proprietary and vary by company). The goal is a useful proxy
signal for the user's review step, not a guarantee of real-world ATS behavior.

## Scoring components
Store in `resume_versions.ats_score` (overall int, e.g. 0-100) and
`resume_versions.ats_score_breakdown` (JSONB) with at least:
- **Keyword overlap**: extract key terms/skills from `job_context.key_requirements` and
  the raw job description, check presence in the tailored resume. List matched and
  missing keywords explicitly — this is the most actionable part of the breakdown for
  the user.
- **Formatting compatibility flags**: things real ATS parsers commonly choke on —
  tables, text boxes, headers/footers containing critical info, non-standard fonts,
  images with embedded text. Since the base template is a fixed docx, this can mostly be
  checked once against the template rather than per-resume, but flag if generated content
  breaks a previously-safe layout (e.g. text overflow forcing a new column).
- **Section completeness**: does the tailored resume actually cover the job's stated
  must-have requirements at all, even if not keyword-identical.

## Implementation approach
- Have the Gemini call return structured JSON for the breakdown (matched keywords, missing
  keywords, notes) rather than free text — makes it directly renderable in the React
  review UI without additional parsing/guessing.
- Keep the keyword extraction step separate and cacheable per job (store in
  `job_context.key_requirements`) so re-tailoring (new `resume_versions` row) doesn't
  redo requirement extraction from scratch — only re-score against the existing list.
- Don't present the score as more authoritative than it is — surface it as "estimated
  match" in the UI, not "ATS pass guarantee."

## To refine later
- Weighting between keyword overlap vs. formatting vs. completeness in the overall score
- Whether to validate against any publicly documented ATS parsing behavior (e.g. known
  Greenhouse/Workday parsing quirks) as the pipeline matures
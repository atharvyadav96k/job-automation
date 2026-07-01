---
name: docx-templating
description: Use this skill whenever filling, generating, or modifying the resume .docx file — any code that takes tailored content (bullet points, summary, skills list) and writes it into the base resume template. Also use when the user wants to change the resume template itself, add new placeholder sections, or fix formatting issues in generated resumes.
---

# Resume DOCX Templating

## Approach
Fill the user's existing `.docx` template rather than generating a resume from scratch.
This preserves their formatting, fonts, and layout choices — the LLM only supplies
content, never layout.

## Recommended library (Go)
Use `github.com/unidoc/unioffice` or `github.com/gomutex/godocx` (evaluate current
maintenance status — pick whichever is actively maintained at implementation time).
Avoid raw XML manipulation of the `.docx` zip unless a library approach fails; it's
fragile and easy to corrupt.

## Placeholder convention
Define clear, greppable placeholders in the base template, e.g.:
- `{{SUMMARY}}`
- `{{SKILLS_LIST}}`
- `{{EXPERIENCE_BULLET_1}}` ... `{{EXPERIENCE_BULLET_N}}` (or a repeating block per job entry)
- `{{PROJECT_TITLE_N}}`, `{{PROJECT_DESC_N}}`

Keep placeholders in a single documented list (e.g. `app/internal/resume/placeholders.go`)
so the LLM prompt and the template stay in sync. If the template changes, update both
together — a mismatch here silently produces broken resumes.

## Common pitfalls
- Track-changes / comments left in the template can break find-and-replace — strip these
  from the base template before use.
- Bullet point lists in Word often use list-style numbering, not literal text bullets —
  replacing text inside a list item should preserve the list formatting, not just swap text.
- Do not let generated content overflow the template's intended length (e.g. a bullet
  point that wraps 3 lines when the template was designed for 1) — validate length before
  writing, or size-check the rendered output.
- Always keep the original base template read-only; write to a copy.

## Testing
Before trusting this in the pipeline, manually open a handful of generated resumes in
Word/LibreOffice to confirm formatting held up — automated tests can check placeholder
replacement happened, but can't easily catch visual breakage.

## To refine later
- Exact placeholder names once the real template is finalized
- Whether repeating sections (multiple jobs/projects) need a loop-based templating
  approach vs a fixed max number of slots
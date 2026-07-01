---
name: resume-file-staging
description: Use this skill for any code that uploads a resume, cover letter, or other generated document to a job application form. Covers the required copy/rename/metadata-clean steps that must happen before any file upload in the browser automation worker. Also use when debugging why an uploaded file looks auto-generated or when a company flags an application as suspicious.
---

# Resume File Staging Before Upload

## Why this exists
Never upload a file straight from its generated path. Generated files often have
tool-identifying names (e.g. `job_4821_v2.docx`) or embedded document metadata that
signals automation. Stage a clean copy first.

## Required steps, in order
1. **Copy** the approved resume PDF (and cover letter, if a separate file) to a temp
   staging directory — never mutate or upload the original generated file. The original
   stays untouched in `resume_versions` for record-keeping.
2. **Rename** using a natural, human filename convention:
   - `FirstName_LastName_Resume.pdf`
   - `FirstName_LastName_Cover_Letter.pdf`
   - Optionally include company name for a per-application-looking file:
     `FirstName_LastName_Resume_CompanyName.pdf`
3. **Clean metadata** — PDF/docx files carry Author, Last Modified By, Producer/Application
   fields. Whatever tool does the docx→PDF conversion (e.g. LibreOffice headless, unoconv)
   will stamp its own name in there by default. Explicitly set:
   - Author → the user's real name (never a fake name — this must stay factual)
   - Strip or overwrite Producer/Application/Creator fields that reveal the conversion tool
4. **Upload from the staged copy only.** Delete staged files after successful submission
   (or on a schedule) — don't let the staging directory grow unbounded.

## Go implementation notes
- For PDF metadata: `github.com/pdfcpu/pdfcpu` supports reading/writing document info
  dictionaries — use this to overwrite Author/Producer/Creator post-conversion.
- Keep the staging function as a single shared utility
  (`app/internal/staging/prepare_upload.go` or similar) called by every place that
  uploads a file — don't duplicate this logic across application flows for different
  ATS platforms.

## Boundary to respect
This is about looking like a normal, human-prepared file — not misrepresenting identity.
Author metadata must be the user's real name. Do not fabricate a different name, company,
or authorship in metadata.

## To refine later
- Whether cover letters should also get a company-specific filename variant
- Retention policy for staged files (how long to keep for debugging vs. cleanup)
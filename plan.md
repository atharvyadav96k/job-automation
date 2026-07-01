# Job Application Automation — Build Plan

## Goal
An automated pipeline that:
1. Searches for relevant jobs matching my skills/experience
2. Tailors my resume (from a base .docx template) for each job, optimized for ATS scoring
3. Generates a matching cover letter
4. Applies automatically on supported platforms (starting with company-direct career pages)
5. Tracks everything so I can review, approve, and monitor results

## Hardware Constraints (must be respected)
- Server: 4GB RAM, 128GB storage, 4-core CPU, old integrated GPU, Docker installed
- **No local LLM inference.** RAM is too tight to run a local model alongside Postgres + Redis + the app reliably. All LLM calls go through an external API.
- Run **one browser automation instance at a time** — headless Chrome uses 200-400MB+; concurrent sessions risk OOM.
- Keep container memory limits explicit in docker-compose to avoid one service starving the others.

## LLM Provider
- **Google Gemini API** (free tier: Flash / Flash-Lite models — generous rate limits, no cost at this usage volume, no credit card required).
- Design the LLM client behind an interface/abstraction in Go so the provider can be swapped later (e.g., to Claude or OpenAI API) without rewriting the pipeline.
- Do not send unnecessary sensitive PII (SSN, phone, exact home address) in prompts — keep those fields only in Postgres, not in LLM calls. Only send what's needed for tailoring: skills, experience bullet points, job description text.

## Tech Stack
- **Go** — orchestration, API server, workers
- **PostgreSQL** — persistent storage (jobs, applications, resume versions, profile)
- **Redis** — job discovery queue, rate limiting, dedup cache
- **chromedp** — browser automation (Go-native, lighter than Playwright for this RAM budget)
- **Docker Compose** — everything containerized, with explicit memory limits per service
- **React** — frontend for profile management UI and pipeline review/detail views (SPA talking to the Go REST API)
  - **MUI (Material UI)** for components (forms, tables, dialogs, tabs) — avoids hand-writing component styling
  - **Tailwind** for layout/spacing utility classes on top — disable Tailwind's `preflight` base reset in its config so it doesn't clobber MUI's own base styles; use Tailwind for spacing/layout only, let MUI own component-level styling
- **docx templating** — Go library (e.g. `unioffice` or template-based `.docx` manipulation) to fill my existing resume template rather than generating from scratch, preserving formatting

---

## Architecture

```
┌─────────────┐     ┌───────────┐     ┌──────────────┐
│ Job Scraper  │────▶│  Redis     │────▶│  Go Worker   │
│ (scheduled)  │     │  Queue     │     │              │
└─────────────┘     └───────────┘     └──────┬───────┘
                                              │
                     ┌────────────────────────┼─────────────────────┐
                     ▼                        ▼                     ▼
              ┌─────────────┐         ┌──────────────┐      ┌──────────────┐
              │ Gemini API   │         │  PostgreSQL   │      │  chromedp     │
              │ (tailor +    │◀───────▶│  (jobs,       │◀────▶│  (apply)      │
              │  ATS score)  │         │  applications,│      │              │
              └─────────────┘         │  resumes)     │      └──────────────┘
                                       └──────────────┘
                                              ▲
                                              │
                                       ┌──────────────┐
                                       │ Review UI     │
                                       │ (simple web   │
                                       │  page or CLI) │
                                       └──────────────┘
```

---

## Data Model (PostgreSQL)

```sql
-- My base profile, skills, experience — source of truth for tailoring
user_profile (
  id, full_name, email, phone, location,
  base_resume_docx_path,      -- path to my original .docx template
  skills JSONB,                -- structured skill list
  experience JSONB,            -- structured work history w/ bullet points
  education JSONB,
  updated_at
)

companies (
  id, name, careers_url, ats_platform,  -- e.g. 'greenhouse', 'lever', 'workday', 'direct'
  notes, created_at
)

jobs (
  id, company_id, title, url, description_raw, description_clean,
  source,                       -- 'scraper', 'api', 'manual'
  discovered_at,
  status                        -- 'new', 'scored', 'skipped', 'queued', 'applied'
)

-- Company-specific context the LLM used, separate from the raw job posting.
-- Kept per-job (not just per-company) since tone/framing can differ posting to posting.
job_context (
  id, job_id,
  company_summary TEXT,         -- inferred: industry, size, stage, culture signals
  key_requirements JSONB,       -- structured list of must-haves extracted from the posting
  inferred_tone TEXT,           -- e.g. 'formal', 'startup-casual' — informs cover letter voice
  research_sources JSONB,       -- URLs/snippets used if company research step is added later
  model_used, created_at
)

-- Every tailoring attempt is kept, not overwritten — lets me compare versions
-- and see exactly how the pipeline's reasoning evolved for a given job.
resume_versions (
  id, job_id, job_context_id,
  version_number INT,           -- increments per job (v1, v2, ...) so I can diff attempts
  generated_resume_docx_path, generated_cover_letter_text,
  ats_score INT,                -- self-estimated match score
  ats_score_breakdown JSONB,    -- e.g. keyword matches found/missing, formatting flags
  changes_summary TEXT,         -- what changed vs the previous version (or vs base resume for v1)
  reasoning TEXT,                -- why these bullets/phrasing were chosen for this job specifically
  model_used, prompt_tokens, completion_tokens,
  created_at,
  approved BOOLEAN DEFAULT FALSE,
  approved_at,
  is_active BOOLEAN DEFAULT TRUE   -- the version actually used for application; only one per job
)

applications (
  id, job_id, resume_version_id,
  method,                       -- 'browser_auto', 'email', 'manual'
  status,                       -- 'pending', 'submitted', 'failed', 'needs_review'
  screenshot_path,
  submitted_at, error_message
)
```

---

## Docker Compose Services

```yaml
services:
  postgres:
    image: postgres:16-alpine
    mem_limit: 512m
    volumes: [pgdata:/var/lib/postgresql/data]

  redis:
    image: redis:7-alpine
    mem_limit: 128m

  app:
    build: ./app          # Go binary: API server + workers
    mem_limit: 1g
    depends_on: [postgres, redis]
    environment:
      - GEMINI_API_KEY=${GEMINI_API_KEY}
      - DATABASE_URL=...
      - REDIS_URL=...

  frontend:
    build: ./frontend     # React app, built and served via nginx (or served as static files by `app`)
    mem_limit: 128m
    depends_on: [app]
    ports:
      - "3000:80"

  browser-worker:
    build: ./browser-worker   # separate container running chromedp + headless Chrome
    mem_limit: 1g
    depends_on: [app]
    # run as its own service so a Chrome crash doesn't take down the API/worker process
```

Total budget: ~2.75GB across containers, leaving headroom on a 4GB host for OS + buffer. The React frontend is a static build served via nginx (cheap, ~128MB), so it doesn't meaningfully eat into the budget. Adjust `mem_limit`s during testing; watch for OOM kills with `docker stats`.

---

## Build Phases

### Phase 0 — Setup
- Docker Compose skeleton with Postgres + Redis running
- Postgres schema migration (tables above)
- Load my base profile (skills, experience, base resume .docx) into `user_profile`
- Gemini API key wired in via env var, test a basic call

### Phase 0.5 — Profile Management Interface
Before the pipeline can tailor good resumes, I need an easy way to keep my profile current — not by editing Postgres rows by hand.
- Simple internal web UI — **React SPA** served as static files, calling the Go REST API — with basic auth (this is personal data + eventually sits on a server with API keys — don't leave it open)
- CRUD screens for:
  - **Skills** — add/edit/remove, optionally tag by category (backend, cloud, tools, etc.)
  - **Projects** — title, description, tech stack, link, bullet points written the way I'd want them to appear on a resume
  - **Work experience** — company, role, dates, bullet points
  - **Education**
  - **Base resume .docx** — upload/replace the template file the pipeline fills in
- Backend: Go REST endpoints (`GET/POST/PUT/DELETE`) on top of the existing `user_profile` tables in the `app` service — no new service needed, just new routes
- This becomes the live source of truth the LLM tailoring step (Phase 2) always reads from — update a project here, and the next matching job's tailored resume picks it up automatically

### Phase 1 — Job Discovery
- Start with 2-3 sources: company career pages on Greenhouse/Lever (public job listing APIs exist for these — lower risk, no login), plus one job board with an API if available
- Scraper/fetcher runs on a schedule (e.g. every few hours), pushes new job IDs into Redis queue, dedup against `jobs` table by URL
- Worker consumes queue, cleans job description HTML → text, stores in `jobs`

### Phase 2 — Tailoring + ATS Scoring
- Worker sends job description + profile data to Gemini
- Prompt asks for: (a) match score, (b) tailored resume bullet points mapped to my existing docx template sections, (c) cover letter, (d) an ATS-style score estimate (keyword overlap with job description, formatting compatibility notes)
- Fill my base .docx template programmatically (preserve formatting, just replace content in known sections/placeholders) rather than generating a resume from scratch
- Store result in `resume_versions`, flagged `approved = false`

### Phase 3 — Review Step (manual gate + detail view)
- Review interface as a **React page** in the same SPA as the profile manager, showing: job title/company, match score, generated resume diff, cover letter
- **Detail view per job**: shows the full pipeline trail — raw job description → extracted `job_context` (company summary, key requirements, inferred tone) → each `resume_versions` entry in order, with `changes_summary` and `reasoning` visible per version, plus ATS score breakdown (which keywords matched/missed)
- If a job was re-tailored (multiple versions), the detail view lets me diff version N against version N-1 to see exactly what changed and why
- I approve or reject each one before anything is applied — approving sets `is_active = true` on the chosen version
- This stays in place until I trust the output quality — not optional early on

### Phase 4 — Browser Automation
- chromedp fills out applications on **company-direct career pages only** to start (Greenhouse/Lever/Workday-hosted forms) — avoid LinkedIn/Indeed Easy Apply due to bot-detection/ToS risk
- **File staging before upload**: never upload the file straight from its generated path/name. Before each application:
  1. Copy the approved resume PDF and cover letter to a temp staging directory
  2. Rename using a natural, human convention — e.g. `FirstName_LastName_Resume.pdf` and `FirstName_LastName_Cover_Letter.pdf` (optionally with company name: `FirstName_LastName_Resume_CompanyName.pdf`, which also reads as attentive to detail rather than automated)
  3. Also check/strip file metadata (docx/PDF properties often embed "Author," "Last Modified By," "Application" fields — e.g. LibreOffice/unoconv or whatever converts docx→PDF may leave tool-identifying metadata). Set Author to my name, strip any generator/tool signature before upload
  4. Upload from the staged, renamed, metadata-cleaned copy — original generated file stays untouched in `resume_versions` for record-keeping
- Fill fields, upload the staged resume PDF, paste cover letter, answer simple screening questions using Gemini-generated answers where fields are dynamic
- Take a screenshot before final submit, store path in `applications`
- **Keep manual "click submit" step initially** — don't auto-submit until form-filling accuracy is proven over real runs

### Phase 5 — Controlled Automation
- Once tailoring + form-filling are reliably accurate, allow auto-submit on a whitelist of low-risk platforms only
- Add randomized delays between actions, one browser session at a time, daily application cap (both for detection risk and to avoid spamming low-quality applications)

### Phase 6 — Tracking Dashboard
- Simple view over `applications`/`jobs`: applied, pending review, response received, rejected
- Use this to refine the base profile and prompt over time based on what's actually landing interviews

---

## Guardrails (keep throughout)
- Human review before submission until proven reliable
- Company-direct pages before third-party platforms (ToS + bot-detection risk is much lower)
- One browser instance at a time — RAM budget
- Daily application cap, even after automation is trusted
- Log everything: job, resume version used, screenshot, timestamp, outcome — for debugging and for learning what's working

## Explicitly Out of Scope (for now)
- LinkedIn/Indeed automated Easy Apply (high ban risk, heavy bot detection)
- Local LLM inference (hardware can't support it well)
- Auto-submit without any review gate (until Phase 5 trust threshold is met)

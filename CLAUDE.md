# Project Rules

## Project Context
Full architecture, data model, and phased build order live in `plan.md` — read it once at
the start of a session, don't re-derive architecture from the codebase. Build one phase
at a time as defined there; don't jump ahead to later phases unless explicitly asked.

## Skills
Project-specific skills live in `.claude/skills/`. View the relevant SKILL.md before
touching related code — don't guess at conventions these already define.
- `.claude/skills/docx-templating/SKILL.md` — filling the resume .docx template
- `.claude/skills/resume-file-staging/SKILL.md` — copy/rename/metadata-strip before any upload
- `.claude/skills/chromedp-application/SKILL.md` — browser automation patterns for job forms
- `.claude/skills/ats-scoring/SKILL.md` — how match score + keyword breakdown is computed

## Codebase Indexing
The codebase-memory-mcp index must stay current with the actual code — a stale index
causes `search_graph`/`get_architecture` to return wrong or missing results, which defeats
the point of using them.
- A git `post-commit` hook (`.git/hooks/post-commit`, or better, a tracked script at
  `scripts/post-commit-index.sh` symlinked into `.git/hooks/` via a setup step) should
  trigger a re-index automatically after every local commit. If this hook doesn't exist
  yet in the repo, set it up before relying on the index for a session.
- If working across multiple commits in one session without the hook active for any
  reason, manually trigger re-indexing before using `search_graph`/`get_architecture`
  again — don't assume the index reflects the latest commit.
- Never skip re-indexing to save time on a task that then reads from the index — reading
  stale results costs more (wrong answers, wasted turns) than the reindex itself.

## Stack
- Go (API server + workers) — `/app`
- React + MUI + Tailwind (frontend SPA) — `/frontend`
- chromedp browser automation — `/browser-worker`
- PostgreSQL, Redis — via Docker Compose, no local install
- Gemini API — LLM calls, key via `GEMINI_API_KEY` env var

## Guidelines
- DO NOT use broad repo-wide searches or long file grep loops.
- Use `search_graph` (codebase-memory-mcp) to find code by concept, symbol, or description before reading any file.
- Use `get_code_snippet` to read specific functions/classes rather than reading entire files.
- Use `get_architecture` to understand project structure before diving into files — prefer this over re-reading `plan.md` mid-task once the session already has it.
- If a file has more than 100 lines, ask the user for specific line numbers rather than reading the whole file.
- When implementing a phase from `plan.md`, quote the specific phase section back before starting, so scope stays contained to that phase.
- Don't modify `docker-compose.yml` memory limits without flagging it — the 4GB host budget is tight and intentional.

## Build & Test Commands
- Go: `cd app && go build ./... && go test ./...`
- Frontend: `cd frontend && npm run build && npm run test`
- Full stack (local): `docker compose up --build`
- Postgres migrations: `cd app && go run ./cmd/migrate`

## Conventions
- Go: standard `gofmt`/`go vet` before considering a task done.
- React: MUI for components, Tailwind for layout/spacing only (preflight disabled in `tailwind.config.js` — don't re-enable it).
- Never commit `.env` or API keys; use `.env.example` as the template.
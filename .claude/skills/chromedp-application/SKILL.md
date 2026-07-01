---
name: chromedp-application
description: Use this skill for any browser automation code — filling job application forms, navigating career pages, handling file uploads via chromedp, or taking screenshots during the apply flow. Use when adding support for a new ATS platform (Greenhouse, Lever, Workday) or debugging why a form fill failed.
---

# chromedp Application Automation

## Scope, in order of rollout
1. Company-direct career pages on **Greenhouse** and **Lever** first — public, well-structured
   forms, no login required, generally lower bot-detection risk and more permissive ToS
   than large platforms.
2. **Workday**-hosted pages next — more complex multi-step forms, higher effort.
3. Do **not** build LinkedIn/Indeed Easy Apply automation — explicitly out of scope per
   `plan.md` due to ToS and bot-detection risk. If asked to add this, flag it back to the
   user rather than implementing silently.

## Resource constraints (server has 4GB RAM)
- Run **one browser instance at a time** — never spawn concurrent chromedp contexts for
  parallel applications. Queue applications and process sequentially.
- Close/cancel the chromedp context explicitly after each application — don't let contexts
  leak across the worker's lifetime.
- Prefer headless mode; only run headed for local debugging.

## Standard flow per application
1. Navigate to the job application URL
2. Wait for form to be interactive (explicit element wait, not fixed `time.Sleep`)
3. Fill known static fields (name, email, phone) from `user_profile`
4. Upload resume/cover letter — **always from the staged, renamed, metadata-cleaned copy**
   (see `resume-file-staging` skill) — use `chromedp.SetUploadFiles` on the file input
5. Handle dynamic screening questions: extract field labels, pass to Gemini for a
   context-appropriate answer, fill in
6. **Screenshot before final submit** — save to `applications.screenshot_path` — this is
   the review artifact, not optional
7. Stop before the actual submit click unless the application has explicit
   `is_active`/approved status and auto-submit is enabled for that platform (Phase 5 gate)

## Detection-risk hygiene
- Add randomized delays between actions (not fixed intervals) — mimics human pacing
- Avoid excessive automation fingerprints where reasonably possible (standard chromedp
  headless Chrome is often sufficient for direct-ATS forms; don't over-engineer stealth
  patches for platforms not in scope)
- Respect the daily application cap defined in `plan.md` — don't let a worker loop bypass it

## Error handling
- On any failure (element not found, timeout, unexpected redirect), mark the application
  `status = 'failed'` with `error_message`, take a screenshot of the failure state, and
  stop — don't retry blindly against a form that changed structure.

## To refine later
- Per-platform selector maps as real forms are encountered (Greenhouse/Lever have fairly
  consistent DOM structure across companies, but not identical)
- Screening-question answer quality review process
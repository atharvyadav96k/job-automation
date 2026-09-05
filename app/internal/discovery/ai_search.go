package discovery

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"job-automation/app/internal/llm"
)

// AISearchSource asks an LLM to find real, currently-open job postings via
// search grounding, instead of a fixed job-board API. Only GroundingSource
// URIs (real search results, not model-authored text) are ever trusted as
// URLs — see llm.GeminiClient.SearchGrounded. Each candidate URL is then
// fetched directly so the stored description comes from the real page, not
// from the model's paraphrase of it; a URL that doesn't resolve (a
// hallucinated-looking link, a dead posting) is simply dropped rather than
// stored as a job.
type AISearchSource struct {
	client     llm.GroundedSearcher
	skillsDesc string
	maxResults int
	httpClient *http.Client
}

func NewAISearchSource(client llm.GroundedSearcher, skillsDesc string, maxResults int) *AISearchSource {
	return &AISearchSource{
		client:     client,
		skillsDesc: skillsDesc,
		maxResults: maxResults,
		httpClient: &http.Client{Timeout: 15 * time.Second},
	}
}

func (s *AISearchSource) Name() string { return "ai-search" }

const aiSearchPrompt = `Search for real, currently open software engineering job postings that
would be a good fit for a candidate with about 1 year of professional experience (NOT senior,
staff, lead, or principal roles) and these skills: %s.

Strongly prefer postings at startups and small/early-stage companies over large enterprises.
Only describe postings you actually found via search just now — do not describe a posting from
memory or invent one. For each posting, mention the company name and job title in your answer
near where you cite it.

Only search for and cite direct links to a single specific job posting — a company's own
careers page listing, or one specific listing on a job board (e.g. a Greenhouse/Lever board, or
a single Indeed/LinkedIn job page). Do NOT cite search-results pages, category/listing pages, or
anything aggregating many jobs at once (e.g. a page titled like "900 Jobs" or "Browse N Jobs").`

// Fetch calls the LLM with search grounding, then fetches every returned
// source URL directly (never trusting the model's own text for the URL).
func (s *AISearchSource) Fetch(ctx context.Context) ([]RawJob, error) {
	prompt := fmt.Sprintf(aiSearchPrompt, s.skillsDesc)
	_, sources, _, err := s.client.SearchGrounded(ctx, prompt)
	if err != nil {
		return nil, fmt.Errorf("grounded search: %w", err)
	}

	limit := s.maxResults
	if limit <= 0 || limit > len(sources) {
		limit = len(sources)
	}

	var jobs []RawJob
	seen := make(map[string]bool)
	for _, src := range sources[:limit] {
		if src.URI == "" || seen[src.URI] {
			continue
		}
		seen[src.URI] = true

		// src.URI is a Google grounding-redirect link, not the real posting
		// URL — resolve it so the stored URL is the actual, stable
		// destination a person could open later.
		finalURL, body, err := s.fetchPage(ctx, src.URI)
		if err != nil {
			continue // dead/unreachable link — drop it rather than store a job with no real content
		}
		if seen[finalURL] {
			continue
		}
		seen[finalURL] = true

		title := extractTitle(body)
		if title == "" {
			title = src.Title
		}
		if title == "" || looksLikeListingPage(title) {
			continue
		}

		jobs = append(jobs, RawJob{
			CompanyName:     companyFromTitle(title, finalURL),
			ATSPlatform:     "ai-search",
			Title:           title,
			URL:             finalURL,
			DescriptionHTML: body,
			Source:          "api",
		})
	}
	return jobs, nil
}

// fetchPage follows redirects (the default http.Client behavior) and
// returns the final resolved URL alongside the page body, since the
// grounding source URI is a redirect link rather than the real destination.
func (s *AISearchSource) fetchPage(ctx context.Context, rawURL string) (finalURL string, body string, err error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return "", "", err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; job-automation-bot/1.0)")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("status %d", resp.StatusCode)
	}

	bodyBytes, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20)) // 2MB cap
	if err != nil {
		return "", "", err
	}
	final := rawURL
	if resp.Request != nil && resp.Request.URL != nil {
		final = resp.Request.URL.String()
	}
	return final, string(bodyBytes), nil
}

var listingPagePatterns = regexp.MustCompile(`(?i)^\s*[\d,]+\+?\s+.*\bjobs\b|\bbrowse\b.*\bjobs\b|\bsearch results\b`)

// looksLikeListingPage catches aggregator search-results/category pages
// (e.g. "900 Entry Level Platform Engineer Jobs | Indeed") that grounding
// sometimes cites instead of a single posting — these aren't a real job to
// tailor a resume for, just a page listing many.
func looksLikeListingPage(title string) bool {
	return listingPagePatterns.MatchString(title)
}

var titleTagRe = regexp.MustCompile(`(?is)<title[^>]*>(.*?)</title>`)

func extractTitle(html string) string {
	m := titleTagRe.FindStringSubmatch(html)
	if len(m) < 2 {
		return ""
	}
	return strings.TrimSpace(CleanHTML(m[1]))
}

// companyFromTitle guesses a company name from a page title using common
// job-posting title patterns ("Role at Company", "Role - Company"). Falls
// back to the URL's host if nothing matches — an approximation, but derived
// from the real fetched page/URL, never invented.
func companyFromTitle(title, rawURL string) string {
	for _, sep := range []string{" at ", " | ", " - "} {
		if idx := strings.LastIndex(title, sep); idx != -1 {
			candidate := strings.TrimSpace(title[idx+len(sep):])
			if candidate != "" {
				return candidate
			}
		}
	}
	if u, err := url.Parse(rawURL); err == nil && u.Host != "" {
		return u.Host
	}
	return "Unknown"
}

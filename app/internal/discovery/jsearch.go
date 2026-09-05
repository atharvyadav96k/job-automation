package discovery

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"time"
)

// JSearchSource searches OpenWeb Ninja's JSearch API (api.openwebninja.com/
// jsearch/search, auth via the x-api-key header — confirmed against a live
// call; this is NOT the RapidAPI-proxy variant, which uses different auth).
// Only ever triggered manually (see SourcesConfig/BuildSources) — it's a
// metered API (free tier: 200 requests/month per key), so it's deliberately
// excluded from the automatic scrape ticker.
//
// apiKeys may hold more than one key so quota can be spread across
// accounts — Fetch tries each in order and falls over to the next on any
// failure (a dedicated account rarely fails for reasons other than quota).
type JSearchSource struct {
	apiKeys    []string
	query      string
	country    string
	limit      int
	httpClient *http.Client
}

func NewJSearchSource(apiKeys []string, query, country string, limit int) *JSearchSource {
	return &JSearchSource{
		apiKeys:    apiKeys,
		query:      query,
		country:    country,
		limit:      limit,
		httpClient: &http.Client{Timeout: 20 * time.Second},
	}
}

func (s *JSearchSource) Name() string { return fmt.Sprintf("jsearch:%s", s.query) }

// jsearchResponse mirrors the fields actually observed in a live response —
// see the discovery.AISearchSource that preceded this for why guessing at
// an external API's schema instead of confirming it is worth avoiding.
type jsearchResponse struct {
	Status string `json:"status"`
	Error  *struct {
		Message string `json:"message"`
	} `json:"error"`
	Data []struct {
		JobTitle        string `json:"job_title"`
		EmployerName    string `json:"employer_name"`
		EmployerWebsite string `json:"employer_website"`
		JobPublisher    string `json:"job_publisher"`
		JobApplyLink    string `json:"job_apply_link"`
		ApplyOptions    []struct {
			ApplyLink string `json:"apply_link"`
			Publisher string `json:"publisher"`
		} `json:"apply_options"`
		JobDescription string `json:"job_description"`
		JobLocation    string `json:"job_location"`
		JobCountry     string `json:"job_country"`
		JobIsRemote    bool   `json:"job_is_remote"`
		JobID          string `json:"job_id"`
	} `json:"data"`
}

func (s *JSearchSource) Fetch(ctx context.Context) ([]RawJob, error) {
	if len(s.apiKeys) == 0 {
		return nil, fmt.Errorf("jsearch: no API keys configured")
	}

	var parsed *jsearchResponse
	var lastErr error
	for i, key := range s.apiKeys {
		p, err := s.searchWithKey(ctx, key)
		if err == nil {
			parsed = p
			break
		}
		lastErr = err
		log.Printf("jsearch: key %d/%d failed (%v), trying next", i+1, len(s.apiKeys), err)
	}
	if parsed == nil {
		return nil, fmt.Errorf("jsearch search %q: all %d key(s) failed, last error: %w", s.query, len(s.apiKeys), lastErr)
	}

	limit := s.limit
	if limit <= 0 || limit > len(parsed.Data) {
		limit = len(parsed.Data)
	}

	jobs := make([]RawJob, 0, limit)
	for _, d := range parsed.Data[:limit] {
		if d.JobApplyLink == "" {
			continue // nothing usable to store as the job's URL
		}

		applyPortal := d.JobPublisher
		if applyPortal == "" && len(d.ApplyOptions) > 0 {
			applyPortal = d.ApplyOptions[0].Publisher
		}

		jobs = append(jobs, RawJob{
			CompanyName:     d.EmployerName,
			ATSPlatform:     "jsearch",
			Title:           d.JobTitle,
			URL:             d.JobApplyLink,
			DescriptionHTML: d.JobDescription,
			Source:          "api",
			ApplyURL:        d.JobApplyLink,
			ApplyPortal:     applyPortal,
			ContactEmail:    extractEmail(d.JobDescription),
		})
	}
	return jobs, nil
}

// searchWithKey makes a single request with one API key. job_requirements=
// under_3_years_experience is the API's own server-side experience-level
// filter (confirmed against the official docs and a live test call) — far
// more reliable than inferring seniority from title/description text after
// the fact.
func (s *JSearchSource) searchWithKey(ctx context.Context, apiKey string) (*jsearchResponse, error) {
	reqURL := fmt.Sprintf(
		"https://api.openwebninja.com/jsearch/search?query=%s&country=%s&page=1&num_pages=1&job_requirements=under_3_years_experience",
		url.QueryEscape(s.query), url.QueryEscape(s.country),
	)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("x-api-key", apiKey)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("status %d", resp.StatusCode)
	}

	var parsed jsearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	if parsed.Status != "OK" {
		msg := parsed.Status
		if parsed.Error != nil && parsed.Error.Message != "" {
			msg = parsed.Error.Message
		}
		return nil, fmt.Errorf("status field %q: %s", parsed.Status, msg)
	}
	return &parsed, nil
}

var emailRe = regexp.MustCompile(`[\w.+-]+@[\w-]+\.[\w.-]+`)

// extractEmail is a best-effort scan of the job description for a literal
// contact email — never guessed/constructed, only ever what's actually
// written in the text.
func extractEmail(description string) string {
	return emailRe.FindString(description)
}

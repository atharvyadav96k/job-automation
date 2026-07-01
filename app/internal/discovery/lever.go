package discovery

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// LeverSource fetches a single company's public postings. Slug is the token
// in jobs.lever.co/{slug} — stored as companies.careers_url for rows with
// ats_platform = 'lever'.
type LeverSource struct {
	CompanyName string
	Slug        string
	httpClient  *http.Client
}

func NewLeverSource(companyName, slug string) *LeverSource {
	return &LeverSource{
		CompanyName: companyName,
		Slug:        slug,
		httpClient:  &http.Client{Timeout: 20 * time.Second},
	}
}

func (s *LeverSource) Name() string { return fmt.Sprintf("lever:%s", s.Slug) }

type leverPosting struct {
	Text        string `json:"text"`
	HostedURL   string `json:"hostedUrl"`
	Description string `json:"description"`
}

func (s *LeverSource) Fetch(ctx context.Context) ([]RawJob, error) {
	url := fmt.Sprintf("https://api.lever.co/v0/postings/%s?mode=json", s.Slug)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("lever board %s: status %d", s.Slug, resp.StatusCode)
	}

	var postings []leverPosting
	if err := json.NewDecoder(resp.Body).Decode(&postings); err != nil {
		return nil, fmt.Errorf("decode lever response: %w", err)
	}

	jobs := make([]RawJob, 0, len(postings))
	for _, p := range postings {
		jobs = append(jobs, RawJob{
			CompanyName:     s.CompanyName,
			ATSPlatform:     "lever",
			Title:           p.Text,
			URL:             p.HostedURL,
			DescriptionHTML: p.Description,
			Source:          "scraper",
		})
	}
	return jobs, nil
}

package discovery

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// GreenhouseSource fetches a single company's public job board. Slug is the
// token in boards.greenhouse.io/{slug} — stored as companies.careers_url
// for rows with ats_platform = 'greenhouse'.
type GreenhouseSource struct {
	CompanyName string
	Slug        string
	httpClient  *http.Client
}

func NewGreenhouseSource(companyName, slug string) *GreenhouseSource {
	return &GreenhouseSource{
		CompanyName: companyName,
		Slug:        slug,
		httpClient:  &http.Client{Timeout: 20 * time.Second},
	}
}

func (s *GreenhouseSource) Name() string { return fmt.Sprintf("greenhouse:%s", s.Slug) }

type greenhouseResponse struct {
	Jobs []struct {
		Title       string `json:"title"`
		AbsoluteURL string `json:"absolute_url"`
		Content     string `json:"content"`
	} `json:"jobs"`
}

func (s *GreenhouseSource) Fetch(ctx context.Context) ([]RawJob, error) {
	url := fmt.Sprintf("https://boards-api.greenhouse.io/v1/boards/%s/jobs?content=true", s.Slug)
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
		return nil, fmt.Errorf("greenhouse board %s: status %d", s.Slug, resp.StatusCode)
	}

	var parsed greenhouseResponse
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return nil, fmt.Errorf("decode greenhouse response: %w", err)
	}

	jobs := make([]RawJob, 0, len(parsed.Jobs))
	for _, j := range parsed.Jobs {
		jobs = append(jobs, RawJob{
			CompanyName:     s.CompanyName,
			ATSPlatform:     "greenhouse",
			Title:           j.Title,
			URL:             j.AbsoluteURL,
			DescriptionHTML: j.Content,
			Source:          "scraper",
		})
	}
	return jobs, nil
}

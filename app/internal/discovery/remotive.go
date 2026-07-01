package discovery

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

// RemotiveSource searches Remotive's free, keyless public job board API.
// It's the "one job board with an API" source from the build plan — used
// until specific companies are added to the companies table for the
// Greenhouse/Lever sources.
type RemotiveSource struct {
	Query      string
	Limit      int
	httpClient *http.Client
}

func NewRemotiveSource(query string, limit int) *RemotiveSource {
	return &RemotiveSource{
		Query:      query,
		Limit:      limit,
		httpClient: &http.Client{Timeout: 20 * time.Second},
	}
}

func (s *RemotiveSource) Name() string { return fmt.Sprintf("remotive:%s", s.Query) }

type remotiveResponse struct {
	Jobs []struct {
		Title       string `json:"title"`
		URL         string `json:"url"`
		CompanyName string `json:"company_name"`
		Description string `json:"description"`
	} `json:"jobs"`
}

func (s *RemotiveSource) Fetch(ctx context.Context) ([]RawJob, error) {
	reqURL := fmt.Sprintf("https://remotive.com/api/remote-jobs?search=%s", url.QueryEscape(s.Query))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, err
	}
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("remotive search %q: status %d", s.Query, resp.StatusCode)
	}

	var parsed remotiveResponse
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return nil, fmt.Errorf("decode remotive response: %w", err)
	}

	limit := s.Limit
	if limit <= 0 || limit > len(parsed.Jobs) {
		limit = len(parsed.Jobs)
	}

	jobs := make([]RawJob, 0, limit)
	for _, j := range parsed.Jobs[:limit] {
		jobs = append(jobs, RawJob{
			CompanyName:     j.CompanyName,
			ATSPlatform:     "remotive",
			Title:           j.Title,
			URL:             j.URL,
			DescriptionHTML: j.Description,
			Source:          "api",
		})
	}
	return jobs, nil
}

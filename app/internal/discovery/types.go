package discovery

import "context"

// RawJob is what a Source produces — enough to dedup and queue, before the
// worker cleans the description and writes the final jobs row.
type RawJob struct {
	CompanyName     string `json:"company_name"`
	ATSPlatform     string `json:"ats_platform"`
	Title           string `json:"title"`
	URL             string `json:"url"`
	DescriptionHTML string `json:"description_html"`
	Source          string `json:"source"` // 'scraper' per jobs.source convention
}

type Source interface {
	Name() string
	Fetch(ctx context.Context) ([]RawJob, error)
}

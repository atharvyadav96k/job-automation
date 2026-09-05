package github

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

const apiBase = "https://api.github.com"

// Repo is the subset of GitHub's repo object this pipeline needs. Topics
// require no special preview header on the current REST API version.
type Repo struct {
	ID          int64    `json:"id"`
	Name        string   `json:"name"`
	HTMLURL     string   `json:"html_url"`
	Description string   `json:"description"`
	Language    string   `json:"language"`
	Topics      []string `json:"topics"`
	Stars       int      `json:"stargazers_count"`
	PushedAt    string   `json:"pushed_at"`
	Fork        bool     `json:"fork"`
}

type Client struct {
	username   string
	token      string
	httpClient *http.Client
}

func NewClient(username, token string) *Client {
	return &Client{username: username, token: token, httpClient: http.DefaultClient}
}

const perPage = 100
const maxPages = 10 // caps at 1000 repos, comfortably above any real account

// ListRepos fetches every repo owned by the configured user, paginating
// until a short page signals the end.
func (c *Client) ListRepos() ([]Repo, error) {
	var all []Repo
	for page := 1; page <= maxPages; page++ {
		reqURL := fmt.Sprintf("%s/users/%s/repos?per_page=%d&page=%d&sort=updated",
			apiBase, url.PathEscape(c.username), perPage, page)

		req, err := http.NewRequest(http.MethodGet, reqURL, nil)
		if err != nil {
			return nil, fmt.Errorf("build request: %w", err)
		}
		req.Header.Set("Accept", "application/vnd.github+json")
		if c.token != "" {
			req.Header.Set("Authorization", "Bearer "+c.token)
		}

		resp, err := c.httpClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf("list repos page %d: %w", page, err)
		}
		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			return nil, fmt.Errorf("list repos page %d: unexpected status %d", page, resp.StatusCode)
		}

		var pageRepos []Repo
		err = json.NewDecoder(resp.Body).Decode(&pageRepos)
		resp.Body.Close()
		if err != nil {
			return nil, fmt.Errorf("decode repos page %d: %w", page, err)
		}

		all = append(all, pageRepos...)
		if len(pageRepos) < perPage {
			break
		}
	}
	return all, nil
}

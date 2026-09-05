package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

const defaultGeminiEndpoint = "https://generativelanguage.googleapis.com/v1beta/models"

type GeminiClient struct {
	apiKey     string
	model      string
	endpoint   string
	httpClient *http.Client
}

// NewGeminiClient builds a Client backed by the Gemini API. Model defaults to
// gemini-2.5-flash (free tier) but can be overridden via GEMINI_MODEL.
func NewGeminiClient(apiKey string) *GeminiClient {
	model := os.Getenv("GEMINI_MODEL")
	if model == "" {
		model = "gemini-2.5-flash"
	}
	return &GeminiClient{
		apiKey:     apiKey,
		model:      model,
		endpoint:   defaultGeminiEndpoint,
		httpClient: &http.Client{Timeout: 60 * time.Second},
	}
}

func (c *GeminiClient) Model() string { return c.model }

type geminiRequest struct {
	Contents         []geminiContent   `json:"contents"`
	GenerationConfig *generationConfig `json:"generationConfig,omitempty"`
	Tools            []geminiTool      `json:"tools,omitempty"`
}

// geminiTool enables Gemini's Google Search grounding — the model's answer
// gets backed by real search results (returned as GroundingMetadata) rather
// than whatever it recalls from training, which is the only way search
// results can be trusted not to be fabricated. Grounding and
// responseMimeType=application/json cannot be used in the same request
// (a Gemini API constraint), so any grounded call must go through
// generateGrounded, never GenerateJSON.
type geminiTool struct {
	GoogleSearch struct{} `json:"google_search"`
}

type generationConfig struct {
	ResponseMimeType string `json:"responseMimeType,omitempty"`
}

type geminiContent struct {
	Parts []geminiPart `json:"parts"`
}

type geminiPart struct {
	Text string `json:"text"`
}

type geminiResponse struct {
	Candidates []struct {
		Content           geminiContent      `json:"content"`
		GroundingMetadata *groundingMetadata `json:"groundingMetadata"`
	} `json:"candidates"`
	UsageMetadata struct {
		PromptTokenCount     int `json:"promptTokenCount"`
		CandidatesTokenCount int `json:"candidatesTokenCount"`
	} `json:"usageMetadata"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}

type groundingMetadata struct {
	GroundingChunks []struct {
		Web struct {
			URI   string `json:"uri"`
			Title string `json:"title"`
		} `json:"web"`
	} `json:"groundingChunks"`
}

// GroundingSource is one real, search-backed citation Gemini's answer was
// grounded on — the URI comes from Google's search index via the grounding
// API, not from the model itself, so it's the one part of a grounded
// response safe to treat as "real", unlike free-text the model writes.
type GroundingSource struct {
	URI   string
	Title string
}

// Usage carries token accounting so callers can persist it (e.g.
// resume_versions.prompt_tokens/completion_tokens).
type Usage struct {
	PromptTokens     int
	CompletionTokens int
}

func (c *GeminiClient) GenerateText(ctx context.Context, prompt string) (string, error) {
	text, _, err := c.generate(ctx, prompt, "")
	return text, err
}

// SearchGrounded answers a prompt using live Google Search grounding. See
// generateGrounded for what is and isn't trustworthy in the result.
func (c *GeminiClient) SearchGrounded(ctx context.Context, prompt string) (string, []GroundingSource, Usage, error) {
	return c.generateGrounded(ctx, prompt)
}

const generateJSONMaxAttempts = 3

// GenerateJSON asks Gemini to respond as JSON (responseMimeType) and
// unmarshals the result into target. Used for structured outputs like the
// tailoring result and ATS score breakdown, so the pipeline never has to
// guess-parse free text.
//
// responseMimeType=application/json does not guarantee syntactically valid
// JSON in practice — Gemini occasionally emits a literal unescaped `"`
// inside a string value (e.g. quoting a phrase for emphasis in a cover
// letter). This is probabilistic per call, not a deterministic prompt bug,
// so a bounded retry is the practical mitigation.
func (c *GeminiClient) GenerateJSON(ctx context.Context, prompt string, target any) (Usage, error) {
	var lastErr error
	var usage Usage
	for attempt := 1; attempt <= generateJSONMaxAttempts; attempt++ {
		text, u, err := c.generate(ctx, prompt, "application/json")
		usage = u
		if err != nil {
			lastErr = err
			continue
		}
		if err := json.Unmarshal([]byte(text), target); err != nil {
			if debugPath := os.Getenv("GEMINI_DEBUG_DUMP_PATH"); debugPath != "" {
				_ = os.WriteFile(debugPath, []byte(text), 0o644)
			}
			lastErr = fmt.Errorf("unmarshal gemini json response: %w (body: %s)", err, text)
			continue
		}
		return usage, nil
	}
	return usage, fmt.Errorf("gemini json generation failed after %d attempts: %w", generateJSONMaxAttempts, lastErr)
}

func (c *GeminiClient) generate(ctx context.Context, prompt, responseMimeType string) (string, Usage, error) {
	reqPayload := geminiRequest{
		Contents: []geminiContent{{Parts: []geminiPart{{Text: prompt}}}},
	}
	if responseMimeType != "" {
		reqPayload.GenerationConfig = &generationConfig{ResponseMimeType: responseMimeType}
	}
	text, _, usage, err := c.call(ctx, reqPayload)
	return text, usage, err
}

// generateGrounded asks Gemini to answer using live Google Search results
// (no responseMimeType — grounding and JSON mode are mutually exclusive on
// this API). The free-text answer itself is still model-authored and can
// misdescribe things, but GroundingSource.URI values come straight from
// Google's search index, not the model, so callers that need to trust a URL
// must use the returned sources rather than any URL mentioned in the text.
func (c *GeminiClient) generateGrounded(ctx context.Context, prompt string) (string, []GroundingSource, Usage, error) {
	reqPayload := geminiRequest{
		Contents: []geminiContent{{Parts: []geminiPart{{Text: prompt}}}},
		Tools:    []geminiTool{{}},
	}
	return c.call(ctx, reqPayload)
}

func (c *GeminiClient) call(ctx context.Context, reqPayload geminiRequest) (string, []GroundingSource, Usage, error) {
	reqBody, err := json.Marshal(reqPayload)
	if err != nil {
		return "", nil, Usage{}, fmt.Errorf("marshal gemini request: %w", err)
	}

	url := fmt.Sprintf("%s/%s:generateContent?key=%s", c.endpoint, c.model, c.apiKey)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(reqBody))
	if err != nil {
		return "", nil, Usage{}, fmt.Errorf("build gemini request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", nil, Usage{}, fmt.Errorf("call gemini api: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", nil, Usage{}, fmt.Errorf("read gemini response: %w", err)
	}

	var parsed geminiResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return "", nil, Usage{}, fmt.Errorf("unmarshal gemini response: %w (body: %s)", err, body)
	}
	if parsed.Error != nil {
		return "", nil, Usage{}, fmt.Errorf("gemini api error: %s", parsed.Error.Message)
	}
	if len(parsed.Candidates) == 0 || len(parsed.Candidates[0].Content.Parts) == 0 {
		return "", nil, Usage{}, fmt.Errorf("gemini response had no candidates (status %d, body: %s)", resp.StatusCode, body)
	}
	usage := Usage{
		PromptTokens:     parsed.UsageMetadata.PromptTokenCount,
		CompletionTokens: parsed.UsageMetadata.CandidatesTokenCount,
	}

	var sources []GroundingSource
	if gm := parsed.Candidates[0].GroundingMetadata; gm != nil {
		for _, chunk := range gm.GroundingChunks {
			if chunk.Web.URI == "" {
				continue
			}
			sources = append(sources, GroundingSource{URI: chunk.Web.URI, Title: chunk.Web.Title})
		}
	}
	return parsed.Candidates[0].Content.Parts[0].Text, sources, usage, nil
}

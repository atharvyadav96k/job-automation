package llm

import "context"

// Client abstracts the LLM provider so it can be swapped (e.g. Claude, OpenAI)
// without touching the pipeline code that calls it.
type Client interface {
	GenerateText(ctx context.Context, prompt string) (string, error)
}

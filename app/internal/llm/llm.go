package llm

import "context"

// Client abstracts the LLM provider so it can be swapped (e.g. Claude, OpenAI)
// without touching the pipeline code that calls it.
type Client interface {
	GenerateText(ctx context.Context, prompt string) (string, error)
	GenerateJSON(ctx context.Context, prompt string, target any) (Usage, error)
}

// GroundedSearcher is an optional capability: not every provider supports
// search grounding, so callers type-assert for it rather than it being part
// of Client (see discovery.AISearchSource).
type GroundedSearcher interface {
	SearchGrounded(ctx context.Context, prompt string) (string, []GroundingSource, Usage, error)
}

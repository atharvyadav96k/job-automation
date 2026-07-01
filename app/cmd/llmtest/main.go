// Phase 0 verification tool: confirms GEMINI_API_KEY is valid and the
// LLM client abstraction can reach the provider. Not part of the pipeline.
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"job-automation/app/internal/config"
	"job-automation/app/internal/llm"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}
	if cfg.GeminiAPIKey == "" {
		log.Fatal("GEMINI_API_KEY is not set")
	}

	client := llm.NewGeminiClient(cfg.GeminiAPIKey)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	reply, err := client.GenerateText(ctx, "Reply with exactly: pong")
	if err != nil {
		log.Fatalf("gemini call failed: %v", err)
	}
	fmt.Printf("gemini reply: %s\n", reply)
}

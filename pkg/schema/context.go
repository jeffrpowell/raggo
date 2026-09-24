package schema

import "time"

// ChunkContext is the LLM-generated passage that situates a chunk within its
// source (document or episode). Shared by both pipelines.
type ChunkContext struct {
	ChunkID       string    `json:"chunk_id"`
	TextHash      string    `json:"text_hash"`
	Context       string    `json:"context"`
	Mode          string    `json:"mode"`
	Model         string    `json:"model"`
	PromptVersion string    `json:"prompt_version"`
	GeneratedAt   time.Time `json:"generated_at"`
}

// Synopsis is a map-reduce summary of a source too large for the context
// model's budget. Cached so reruns don't regenerate it.
type Synopsis struct {
	SourceID      string    `json:"source_id"`
	Text          string    `json:"text"`
	Model         string    `json:"model"`
	PromptVersion string    `json:"prompt_version"`
	BudgetChars   int       `json:"budget_chars"`
	GeneratedAt   time.Time `json:"generated_at"`
}

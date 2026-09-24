package contextual

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/jeffrpowell/raggo/pkg/logging"
	"github.com/jeffrpowell/raggo/pkg/schema"
	"github.com/jeffrpowell/raggo/pkg/storage"
)

// Job is one source (document or episode) to contextualize.
type Job struct {
	SourceID string
	Meta     Meta
	Pieces   []Piece
}

func ContextPath(contextDir, sourceID string) string {
	return filepath.Join(contextDir, fmt.Sprintf("%s.jsonl", sourceID))
}

func SynopsisPath(contextDir, sourceID string) string {
	return filepath.Join(contextDir, fmt.Sprintf("%s.synopsis.json", sourceID))
}

// PendingIDs lists source IDs that have a chunk file but no context file.
func PendingIDs(chunkDir, contextDir string) ([]string, error) {
	files, err := filepath.Glob(filepath.Join(chunkDir, "*.jsonl"))
	if err != nil {
		return nil, err
	}
	var ids []string
	for _, f := range files {
		id := strings.TrimSuffix(filepath.Base(f), ".jsonl")
		if !storage.MarkerExists(ContextPath(contextDir, id)) {
			ids = append(ids, id)
		}
	}
	return ids, nil
}

// RunJob contextualizes one source and writes contexts/<id>.jsonl atomically
// so a partially processed source is never mistaken for a finished one.
func RunJob(ctx context.Context, log *logging.Logger, c Completer, model string, job Job, contextDir string, budgetChars int) error {
	contextPath := ContextPath(contextDir, job.SourceID)
	if storage.MarkerExists(contextPath) {
		log.Info("Contexts already exist: %s", contextPath)
		return nil
	}
	if len(job.Pieces) == 0 {
		log.Warn("No chunks for %s, skipping", job.SourceID)
		return nil
	}

	mode := Mode(job.Pieces, budgetChars)
	log.Info("Contextualizing %s: %d chunks, mode=%s, budget=%d chars", job.SourceID, len(job.Pieces), mode, budgetChars)

	synopsis := ""
	if mode == ModeWindowed {
		var err error
		synopsis, err = loadOrBuildSynopsis(ctx, log, c, model, job, contextDir, budgetChars)
		if err != nil {
			return err
		}
	}

	results, err := Contextualize(ctx, c, job.Meta, job.Pieces, synopsis, budgetChars, func(done, total int) {
		if done%10 == 0 {
			log.Info("Contextualized %d/%d chunks", done, total)
		}
	})
	if err != nil {
		return err
	}

	tmpPath := contextPath + ".tmp"
	if err := os.Remove(tmpPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove stale temp file: %w", err)
	}
	now := time.Now()
	for _, r := range results {
		if err := storage.AppendJSONL(tmpPath, schema.ChunkContext{
			ChunkID:       r.ChunkID,
			TextHash:      r.TextHash,
			Context:       r.Context,
			Mode:          r.Mode,
			Model:         model,
			PromptVersion: PromptVersion,
			GeneratedAt:   now,
		}); err != nil {
			return fmt.Errorf("append context: %w", err)
		}
	}
	if err := os.Rename(tmpPath, contextPath); err != nil {
		return fmt.Errorf("finalize contexts: %w", err)
	}

	log.Info("Wrote %d contexts to: %s", len(results), contextPath)
	return nil
}

func loadOrBuildSynopsis(ctx context.Context, log *logging.Logger, c Completer, model string, job Job, contextDir string, budgetChars int) (string, error) {
	path := SynopsisPath(contextDir, job.SourceID)
	var cached schema.Synopsis
	if err := storage.ReadJSON(path, &cached); err == nil && cached.PromptVersion == PromptVersion && cached.Text != "" {
		log.Info("Using cached synopsis: %s", path)
		return cached.Text, nil
	}

	log.Info("Building synopsis for %s", job.SourceID)
	text, err := BuildSynopsis(ctx, c, job.Meta, job.Pieces, budgetChars)
	if err != nil {
		return "", fmt.Errorf("build synopsis: %w", err)
	}
	if err := storage.WriteJSON(path, schema.Synopsis{
		SourceID:      job.SourceID,
		Text:          text,
		Model:         model,
		PromptVersion: PromptVersion,
		BudgetChars:   budgetChars,
		GeneratedAt:   time.Now(),
	}); err != nil {
		return "", fmt.Errorf("write synopsis: %w", err)
	}
	return text, nil
}

// LoadContexts reads contexts/<id>.jsonl into a map keyed by chunk ID.
// A missing file returns an empty map so callers fall back to raw chunks.
func LoadContexts(contextDir, sourceID string) (map[string]schema.ChunkContext, error) {
	contexts := map[string]schema.ChunkContext{}
	if contextDir == "" {
		return contexts, nil
	}
	path := ContextPath(contextDir, sourceID)
	if !storage.MarkerExists(path) {
		return contexts, nil
	}
	err := storage.ReadJSONL(path, func(line []byte) error {
		var cc schema.ChunkContext
		if err := json.Unmarshal(line, &cc); err != nil {
			return err
		}
		contexts[cc.ChunkID] = cc
		return nil
	})
	return contexts, err
}

// Lookup returns the context for a chunk, or "" when none exists or the chunk
// text changed since the context was generated.
func Lookup(contexts map[string]schema.ChunkContext, chunkID, textHash string) string {
	cc, ok := contexts[chunkID]
	if !ok || cc.TextHash != textHash {
		return ""
	}
	return cc.Context
}

// EmbeddingText is the text embedded and BM25-indexed for a chunk.
func EmbeddingText(chunkContext, text string) string {
	if chunkContext == "" {
		return text
	}
	return chunkContext + "\n\n" + text
}

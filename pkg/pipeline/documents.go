package pipeline

import (
	"fmt"

	"github.com/jeffrpowell/raggo/pkg/config"
)

func BuildDocumentPipeline(cfg *config.Config, corpus config.CorpusConfig) *Pipeline {
	return &Pipeline{
		Stages: []Stage{
			{
				Name:   "scan",
				Binary: "/usr/local/bin/raggo-scan-documents",
				BuildArgs: func(ctx *ExecutionContext) []string {
					rootPath := corpus.Config["root_path"].(string)
					return []string{
						"-config", "/etc/raggo/raggo.yml",
						"-corpus-id", corpus.ID,
						"-root-path", rootPath,
						"-manifest-dir", cfg.Storage.Documents.Text,
					}
				},
				CheckSkip: func(ctx *ExecutionContext) (bool, error) {
					return false, nil
				},
			},
			{
				Name:   "extract",
				Binary: "/usr/local/bin/raggo-extract-text",
				BuildArgs: func(ctx *ExecutionContext) []string {
					return []string{
						"-config", "/etc/raggo/raggo.yml",
						"-corpus-id", corpus.ID,
						"-text-dir", cfg.Storage.Documents.Text,
					}
				},
				Items:    documentManifestItems(cfg.Storage.Documents.Text),
				ItemFlag: "-document-id",
				CheckSkip: func(ctx *ExecutionContext) (bool, error) {
					return false, nil
				},
			},
			{
				Name:   "chunk",
				Binary: "/usr/local/bin/raggo-chunk-text",
				BuildArgs: func(ctx *ExecutionContext) []string {
					chunkSize := 1000
					overlapSize := 100
					if cs, ok := corpus.Config["chunk_size"].(int); ok {
						chunkSize = cs
					}
					if os, ok := corpus.Config["overlap_size"].(int); ok {
						overlapSize = os
					}
					return []string{
						"-config", "/etc/raggo/raggo.yml",
						"-corpus-id", corpus.ID,
						"-text-dir", cfg.Storage.Documents.Text,
						"-chunk-dir", cfg.Storage.Documents.Chunks,
						"-chunk-size", fmt.Sprintf("%d", chunkSize),
						"-overlap-size", fmt.Sprintf("%d", overlapSize),
					}
				},
				Items:    idsFromFiles(cfg.Storage.Documents.Text, ".json"),
				ItemFlag: "-document-id",
				CheckSkip: func(ctx *ExecutionContext) (bool, error) {
					return false, nil
				},
			},
			{
				Name:   "contextualize",
				Binary: "/usr/local/bin/raggo-contextualize-text",
				BuildArgs: func(ctx *ExecutionContext) []string {
					args := []string{
						"-config", "/etc/raggo/raggo.yml",
						"-corpus-id", corpus.ID,
						"-manifest-dir", cfg.Storage.Documents.Text,
						"-chunk-dir", cfg.Storage.Documents.Chunks,
						"-context-dir", cfg.Storage.Documents.Contexts,
					}
					if budget, ok := corpus.Config["context_budget_chars"].(int); ok {
						args = append(args, "-context-budget-chars", fmt.Sprintf("%d", budget))
					}
					return args
				},
				CheckSkip: func(ctx *ExecutionContext) (bool, error) {
					return !contextualizeEnabled(corpus), nil
				},
			},
			{
				Name:   "embed",
				Binary: "/usr/local/bin/raggo-embed-text",
				BuildArgs: func(ctx *ExecutionContext) []string {
					return []string{
						"-config", "/etc/raggo/raggo.yml",
						"-corpus-id", corpus.ID,
						"-chunk-dir", cfg.Storage.Documents.Chunks,
						"-context-dir", cfg.Storage.Documents.Contexts,
						"-embedding-dir", cfg.Storage.Documents.Embeddings,
					}
				},
				Items:    idsFromFiles(cfg.Storage.Documents.Chunks, ".jsonl"),
				ItemFlag: "-document-id",
				CheckSkip: func(ctx *ExecutionContext) (bool, error) {
					return false, nil
				},
			},
			{
				Name:   "index",
				Binary: "/usr/local/bin/raggo-index-documents",
				BuildArgs: func(ctx *ExecutionContext) []string {
					collection := corpus.Config["collection"].(string)
					corpusName := corpus.Config["corpus"].(string)
					return []string{
						"-config", "/etc/raggo/raggo.yml",
						"-corpus-id", corpus.ID,
						"-chunk-dir", cfg.Storage.Documents.Chunks,
						"-context-dir", cfg.Storage.Documents.Contexts,
						"-embedding-dir", cfg.Storage.Documents.Embeddings,
						"-index-dir", cfg.Storage.Documents.Index,
						"-collection", collection,
						"-corpus", corpusName,
					}
				},
				Items:    idsFromFiles(cfg.Storage.Documents.Chunks, ".jsonl"),
				ItemFlag: "-document-id",
				CheckSkip: func(ctx *ExecutionContext) (bool, error) {
					return false, nil
				},
			},
		},
	}
}

// contextualizeEnabled reports whether a corpus opts into contextual
// retrieval. Defaults to true; set `contextualize: false` to skip the stage.
func contextualizeEnabled(corpus config.CorpusConfig) bool {
	if enabled, ok := corpus.Config["contextualize"].(bool); ok {
		return enabled
	}
	return true
}

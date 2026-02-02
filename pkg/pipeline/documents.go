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
				CheckSkip: func(ctx *ExecutionContext) (bool, error) {
					return false, nil
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
						"-embedding-dir", cfg.Storage.Documents.Embeddings,
					}
				},
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
						"-embedding-dir", cfg.Storage.Documents.Embeddings,
						"-index-dir", cfg.Storage.Documents.Index,
						"-collection", collection,
						"-corpus", corpusName,
					}
				},
				CheckSkip: func(ctx *ExecutionContext) (bool, error) {
					return false, nil
				},
			},
		},
	}
}

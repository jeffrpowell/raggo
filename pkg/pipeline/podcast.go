package pipeline

import (
	"fmt"
	"path/filepath"

	"github.com/jeffrpowell/raggo/pkg/config"
)

func BuildPodcastPipeline(cfg *config.Config, corpus config.CorpusConfig) *Pipeline {
	return &Pipeline{
		Stages: []Stage{
			{
				Name:   "rss",
				Binary: "/usr/local/bin/raggo-rss-podcast",
				BuildArgs: func(ctx *ExecutionContext) []string {
					feedURL := corpus.Config["feed_url"].(string)
					return []string{
						"-config", "/etc/raggo/raggo.yml",
						"-corpus-id", corpus.ID,
						"-feed-url", feedURL,
						"-raw-dir", cfg.Storage.Podcast.Audio,
						"-manifest-dir", cfg.Storage.Podcast.Audio,
					}
				},
				CheckSkip: func(ctx *ExecutionContext) (bool, error) {
					return false, nil
				},
			},
			{
				Name:   "audio",
				Binary: "/usr/local/bin/raggo-download-audio",
				BuildArgs: func(ctx *ExecutionContext) []string {
					feedURL := corpus.Config["feed_url"].(string)
					workers := 4
					if w, ok := corpus.Config["workers"].(int); ok {
						workers = w
					}
					manifestPath := filepath.Join(cfg.Storage.Podcast.Audio, fmt.Sprintf("%s.jsonl", hashString(feedURL)))
					return []string{
						"-config", "/etc/raggo/raggo.yml",
						"-corpus-id", corpus.ID,
						"-manifest", manifestPath,
						"-audio-dir", cfg.Storage.Podcast.Audio,
						"-workers", fmt.Sprintf("%d", workers),
					}
				},
				CheckSkip: func(ctx *ExecutionContext) (bool, error) {
					return false, nil
				},
			},
			{
				Name:   "stt",
				Binary: "/usr/local/bin/raggo-stt-audio",
				BuildArgs: func(ctx *ExecutionContext) []string {
					return []string{
						"-config", "/etc/raggo/raggo.yml",
						"-corpus-id", corpus.ID,
						"-transcript-dir", cfg.Storage.Podcast.Transcripts,
					}
				},
				Items:    audioItems(cfg.Storage.Podcast.Audio),
				ItemFlag: "-episode-id",
				CheckSkip: func(ctx *ExecutionContext) (bool, error) {
					return false, nil
				},
			},
			{
				Name:   "normalize",
				Binary: "/usr/local/bin/raggo-normalize-podcast",
				BuildArgs: func(ctx *ExecutionContext) []string {
					return []string{
						"-config", "/etc/raggo/raggo.yml",
						"-corpus-id", corpus.ID,
						"-transcript-dir", cfg.Storage.Podcast.Transcripts,
					}
				},
				Items:    idsFromFiles(cfg.Storage.Podcast.Transcripts, ".json", ".normalized.json"),
				ItemFlag: "-episode-id",
				CheckSkip: func(ctx *ExecutionContext) (bool, error) {
					return false, nil
				},
			},
			{
				Name:   "chunk",
				Binary: "/usr/local/bin/raggo-chunk-podcast",
				BuildArgs: func(ctx *ExecutionContext) []string {
					return []string{
						"-config", "/etc/raggo/raggo.yml",
						"-corpus-id", corpus.ID,
						"-transcript-dir", cfg.Storage.Podcast.Transcripts,
						"-chunk-dir", cfg.Storage.Podcast.Chunks,
					}
				},
				Items:    idsFromFiles(cfg.Storage.Podcast.Transcripts, ".normalized.json"),
				ItemFlag: "-episode-id",
				CheckSkip: func(ctx *ExecutionContext) (bool, error) {
					return false, nil
				},
			},
			{
				Name:   "contextualize",
				Binary: "/usr/local/bin/raggo-contextualize-podcast",
				BuildArgs: func(ctx *ExecutionContext) []string {
					args := []string{
						"-config", "/etc/raggo/raggo.yml",
						"-corpus-id", corpus.ID,
						"-manifest-dir", cfg.Storage.Podcast.Audio,
						"-chunk-dir", cfg.Storage.Podcast.Chunks,
						"-context-dir", cfg.Storage.Podcast.Contexts,
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
				Binary: "/usr/local/bin/raggo-embed-podcast",
				BuildArgs: func(ctx *ExecutionContext) []string {
					return []string{
						"-config", "/etc/raggo/raggo.yml",
						"-corpus-id", corpus.ID,
						"-chunk-dir", cfg.Storage.Podcast.Chunks,
						"-context-dir", cfg.Storage.Podcast.Contexts,
						"-embedding-dir", cfg.Storage.Podcast.Embeddings,
					}
				},
				Items:    idsFromFiles(cfg.Storage.Podcast.Chunks, ".jsonl"),
				ItemFlag: "-episode-id",
				CheckSkip: func(ctx *ExecutionContext) (bool, error) {
					return false, nil
				},
			},
			{
				Name:   "index",
				Binary: "/usr/local/bin/raggo-index-podcast",
				BuildArgs: func(ctx *ExecutionContext) []string {
					collection := corpus.Config["collection"].(string)
					corpusName := corpus.Config["corpus"].(string)
					return []string{
						"-config", "/etc/raggo/raggo.yml",
						"-corpus-id", corpus.ID,
						"-chunk-dir", cfg.Storage.Podcast.Chunks,
						"-context-dir", cfg.Storage.Podcast.Contexts,
						"-embedding-dir", cfg.Storage.Podcast.Embeddings,
						"-index-dir", cfg.Storage.Podcast.Index,
						"-collection", collection,
						"-corpus", corpusName,
					}
				},
				Items:    idsFromFiles(cfg.Storage.Podcast.Chunks, ".jsonl"),
				ItemFlag: "-episode-id",
				CheckSkip: func(ctx *ExecutionContext) (bool, error) {
					return false, nil
				},
			},
		},
	}
}

func hashString(s string) string {
	return fmt.Sprintf("%x", []byte(s)[:16])
}

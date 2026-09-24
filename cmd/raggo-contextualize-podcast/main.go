package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/jeffrpowell/raggo/pkg/config"
	"github.com/jeffrpowell/raggo/pkg/contextual"
	"github.com/jeffrpowell/raggo/pkg/llm"
	"github.com/jeffrpowell/raggo/pkg/logging"
	"github.com/jeffrpowell/raggo/pkg/schema"
	"github.com/jeffrpowell/raggo/pkg/storage"
)

var errFound = errors.New("found")

func main() {
	var (
		configPath      string
		corpusID        string
		episodeID       string
		manifestDir     string
		chunkDir        string
		contextDir      string
		contextEndpoint string
		contextModel    string
		budgetChars     int
		maxTokens       int
		disableThinking bool
	)

	flag.StringVar(&configPath, "config", "", "Path to raggo.yml config file")
	flag.StringVar(&corpusID, "corpus-id", "", "Corpus ID from config")
	flag.StringVar(&episodeID, "episode-id", "", "Episode ID (omit to process every chunk file without contexts)")
	flag.StringVar(&manifestDir, "manifest-dir", "", "Directory for episode manifests and feed metadata")
	flag.StringVar(&chunkDir, "chunk-dir", "", "Directory for chunks")
	flag.StringVar(&contextDir, "context-dir", "", "Directory for chunk contexts")
	flag.StringVar(&contextEndpoint, "context-endpoint", "", "OpenAI-compatible chat completions endpoint (empty disables contextualization)")
	flag.StringVar(&contextModel, "context-model", "", "Context model name")
	flag.IntVar(&budgetChars, "context-budget-chars", 0, "Max characters of transcript text per prompt")
	flag.IntVar(&maxTokens, "max-tokens", 150, "Max tokens per generated context")
	flag.BoolVar(&disableThinking, "disable-thinking", true, "Send chat_template_kwargs.enable_thinking=false (Qwen3.x)")
	flag.Parse()

	log := logging.New("raggo-contextualize-podcast")

	var cfg *config.Config
	if configPath != "" {
		var err error
		cfg, err = config.Load(configPath)
		if err != nil {
			log.Fatal("Failed to load config: %v", err)
		}
	}

	manifestDir = config.ResolveAudioDir(cfg, corpusID, manifestDir)
	chunkDir = config.ResolveChunksDir(cfg, corpusID, chunkDir)
	contextDir = config.ResolvePodcastContextsDir(cfg, corpusID, contextDir)
	contextEndpoint = config.ResolveContextEndpoint(cfg, contextEndpoint)
	contextModel = config.ResolveContextModel(cfg, contextModel)
	budgetChars = config.ResolveContextBudgetChars(cfg, budgetChars)

	if contextEndpoint == "" {
		log.Warn("No context endpoint configured, skipping contextualization")
		return
	}

	ids := []string{episodeID}
	if episodeID == "" {
		var err error
		ids, err = contextual.PendingIDs(chunkDir, contextDir)
		if err != nil {
			log.Fatal("Failed to list pending episodes: %v", err)
		}
		log.Info("Found %d episodes pending contextualization", len(ids))
	}

	client := llm.NewClient(contextEndpoint, contextModel, maxTokens, disableThinking)
	for _, id := range ids {
		if err := run(log, client, id, manifestDir, chunkDir, contextDir, budgetChars); err != nil {
			log.Fatal("Failed %s: %v", id, err)
		}
	}
}

func run(log *logging.Logger, client *llm.Client, episodeID, manifestDir, chunkDir, contextDir string, budgetChars int) error {
	chunkPath := filepath.Join(chunkDir, fmt.Sprintf("%s.jsonl", episodeID))

	var pieces []contextual.Piece
	if err := storage.ReadJSONL(chunkPath, func(line []byte) error {
		var chunk schema.Chunk
		if err := json.Unmarshal(line, &chunk); err != nil {
			return err
		}
		pieces = append(pieces, contextual.Piece{ID: chunk.ChunkID, Text: chunk.Text, TextHash: chunk.TextHash})
		return nil
	}); err != nil {
		return fmt.Errorf("read chunks: %w", err)
	}

	meta := contextual.Meta{}
	episode, feed, err := findEpisode(manifestDir, episodeID)
	if err != nil {
		log.Warn("Could not load episode manifest: %v", err)
	} else {
		meta.Title = episode.Title
		meta.Description = episode.Description
		if feed != nil {
			meta.Source = "Podcast: " + feed.Title
		}
	}

	job := contextual.Job{SourceID: episodeID, Meta: meta, Pieces: pieces}
	return contextual.RunJob(context.Background(), log, client, client.Model(), job, contextDir, budgetChars)
}

// findEpisode scans <feedHash>.jsonl manifests for the episode and loads the
// matching <feedHash>.json feed metadata when present.
func findEpisode(manifestDir, episodeID string) (*schema.EpisodeManifest, *schema.PodcastFeed, error) {
	files, err := filepath.Glob(filepath.Join(manifestDir, "*.jsonl"))
	if err != nil {
		return nil, nil, err
	}

	for _, file := range files {
		var result *schema.EpisodeManifest
		err := storage.ReadJSONL(file, func(line []byte) error {
			var manifest schema.EpisodeManifest
			if err := json.Unmarshal(line, &manifest); err != nil {
				return nil
			}
			if manifest.EpisodeID == episodeID {
				result = &manifest
				return errFound
			}
			return nil
		})
		if errors.Is(err, errFound) {
			var feed schema.PodcastFeed
			if err := storage.ReadJSON(strings.TrimSuffix(file, ".jsonl")+".json", &feed); err != nil {
				return result, nil, nil
			}
			return result, &feed, nil
		}
		if err != nil {
			return nil, nil, err
		}
	}

	return nil, nil, fmt.Errorf("episode manifest not found")
}

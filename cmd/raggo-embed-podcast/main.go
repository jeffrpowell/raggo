package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/jeffrpowell/raggo/pkg/config"
	"github.com/jeffrpowell/raggo/pkg/contextual"
	"github.com/jeffrpowell/raggo/pkg/hashing"
	"github.com/jeffrpowell/raggo/pkg/logging"
	"github.com/jeffrpowell/raggo/pkg/schema"
	"github.com/jeffrpowell/raggo/pkg/storage"
)

type EmbedRequest struct {
	Text  string `json:"text"`
	Model string `json:"model"`
}

type EmbedResponse struct {
	Embedding []float32 `json:"embedding"`
	Model     string    `json:"model"`
	Version   string    `json:"version"`
}

func main() {
	var (
		configPath      string
		corpusID        string
		episodeID       string
		chunkDir        string
		contextDir      string
		embeddingDir    string
		embedEndpoint   string
		model           string
		modelVersion    string
	)

	flag.StringVar(&configPath, "config", "", "Path to raggo.yml config file")
	flag.StringVar(&corpusID, "corpus-id", "", "Corpus ID from config")
	flag.StringVar(&episodeID, "episode-id", "", "Episode ID")
	flag.StringVar(&chunkDir, "chunk-dir", "data/chunks", "Directory for chunks")
	flag.StringVar(&contextDir, "context-dir", "", "Directory for chunk contexts (embeds context + chunk when present)")
	flag.StringVar(&embeddingDir, "embedding-dir", "data/embeddings", "Directory for embeddings")
	flag.StringVar(&embedEndpoint, "embed-endpoint", "", "Embedding service HTTP endpoint (optional)")
	flag.StringVar(&model, "model", "text-embedding-3-small", "Embedding model name")
	flag.StringVar(&modelVersion, "model-version", "v1", "Embedding model version")
	flag.Parse()

	log := logging.New("raggo-embed-podcast")

	var cfg *config.Config
	if configPath != "" {
		var err error
		cfg, err = config.Load(configPath)
		if err != nil {
			log.Fatal("Failed to load config: %v", err)
		}
	}

	chunkDir = config.ResolveChunksDir(cfg, corpusID, chunkDir)
	contextDir = config.ResolvePodcastContextsDir(cfg, corpusID, contextDir)
	embeddingDir = config.ResolveEmbeddingsDir(cfg, corpusID, embeddingDir)
	embedEndpoint = config.ResolveEmbedEndpoint(cfg, embedEndpoint)

	if episodeID == "" {
		log.Fatal("episode-id is required")
	}

	if err := run(log, episodeID, chunkDir, contextDir, embeddingDir, embedEndpoint, model, modelVersion); err != nil {
		log.Fatal("Failed: %v", err)
	}
}

func run(log *logging.Logger, episodeID, chunkDir, contextDir, embeddingDir, embedEndpoint, model, modelVersion string) error {
	chunkPath := filepath.Join(chunkDir, fmt.Sprintf("%s.jsonl", episodeID))
	embeddingPath := filepath.Join(embeddingDir, fmt.Sprintf("%s.jsonl", episodeID))

	if storage.MarkerExists(embeddingPath) {
		log.Info("Embeddings already exist: %s", embeddingPath)
		return nil
	}

	log.Info("Loading chunks: %s", chunkPath)

	var chunks []schema.Chunk
	if err := storage.ReadJSONL(chunkPath, func(line []byte) error {
		var chunk schema.Chunk
		if err := json.Unmarshal(line, &chunk); err != nil {
			return err
		}
		chunks = append(chunks, chunk)
		return nil
	}); err != nil {
		return fmt.Errorf("read chunks: %w", err)
	}

	contexts, err := contextual.LoadContexts(contextDir, episodeID)
	if err != nil {
		return fmt.Errorf("read contexts: %w", err)
	}

	log.Info("Generating embeddings for %d chunks (%d with context)", len(chunks), len(contexts))

	if err := os.Remove(embeddingPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove old embeddings: %w", err)
	}

	for i, chunk := range chunks {
		var vector []float32
		var err error
		chunkContext := contextual.Lookup(contexts, chunk.ChunkID, chunk.TextHash)

		if embedEndpoint != "" {
			vector, err = generateEmbeddingViaHTTP(embedEndpoint, contextual.EmbeddingText(chunkContext, chunk.Text), model)
			if err != nil {
				return fmt.Errorf("generate embedding for chunk %d: %w", i, err)
			}
		} else {
			log.Warn("No embedding endpoint configured, generating placeholder")
			vector = generatePlaceholderEmbedding()
		}

		embedding := schema.Embedding{
			ChunkID:      chunk.ChunkID,
			EpisodeID:    chunk.EpisodeID,
			ChunkHash:    chunk.TextHash,
			ContextHash:  contextHash(chunkContext),
			Model:        model,
			ModelVersion: modelVersion,
			Vector:       vector,
			Dimension:    len(vector),
			GeneratedAt:  time.Now(),
		}

		if err := storage.AppendJSONL(embeddingPath, embedding); err != nil {
			return fmt.Errorf("append embedding: %w", err)
		}

		if (i+1)%10 == 0 {
			log.Info("Processed %d/%d chunks", i+1, len(chunks))
		}
	}

	log.Info("Wrote %d embeddings to: %s", len(chunks), embeddingPath)
	return nil
}

func generateEmbeddingViaHTTP(endpoint, text, model string) ([]float32, error) {
	reqBody := EmbedRequest{
		Text:  text,
		Model: model,
	}
	data, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	resp, err := http.Post(endpoint, "application/json", bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("http %d: %s", resp.StatusCode, string(body))
	}

	var embedResp EmbedResponse
	if err := json.NewDecoder(resp.Body).Decode(&embedResp); err != nil {
		return nil, err
	}

	return embedResp.Embedding, nil
}

func generatePlaceholderEmbedding() []float32 {
	vec := make([]float32, 384)
	for i := range vec {
		vec[i] = 0.1
	}
	return vec
}

func contextHash(chunkContext string) string {
	if chunkContext == "" {
		return ""
	}
	return hashing.HashString(chunkContext)
}

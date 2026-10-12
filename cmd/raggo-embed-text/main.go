package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/jeffrpowell/raggo/pkg/config"
	"github.com/jeffrpowell/raggo/pkg/contextual"
	"github.com/jeffrpowell/raggo/pkg/embed"
	"github.com/jeffrpowell/raggo/pkg/hashing"
	"github.com/jeffrpowell/raggo/pkg/logging"
	"github.com/jeffrpowell/raggo/pkg/schema"
	"github.com/jeffrpowell/raggo/pkg/storage"
)

func main() {
	var (
		configPath      string
		corpusID        string
		documentID      string
		chunkDir        string
		contextDir      string
		embeddingDir    string
		embedEndpoint   string
		embedAPI        string
		model           string
		modelVersion    string
	)

	flag.StringVar(&configPath, "config", "", "Path to raggo.yml config file")
	flag.StringVar(&corpusID, "corpus-id", "", "Corpus ID from config")
	flag.StringVar(&documentID, "document-id", "", "Document ID")
	flag.StringVar(&chunkDir, "chunk-dir", "", "Directory for chunks (default: from config)")
	flag.StringVar(&contextDir, "context-dir", "", "Directory for chunk contexts (embeds context + chunk when present)")
	flag.StringVar(&embeddingDir, "embedding-dir", "", "Directory for embeddings (default: from config)")
	flag.StringVar(&embedEndpoint, "embed-endpoint", "", "Embedding service HTTP endpoint (optional)")
	flag.StringVar(&embedAPI, "embed-api", "", "Embedding request format: openai or raggo (default openai)")
	flag.StringVar(&model, "model", "", "Embedding model name (default text-embedding-3-small)")
	flag.StringVar(&modelVersion, "model-version", "v1", "Embedding model version")
	flag.Parse()

	log := logging.New("raggo-embed-text")

	var cfg *config.Config
	if configPath != "" {
		var err error
		cfg, err = config.Load(configPath)
		if err != nil {
			log.Fatal("Failed to load config: %v", err)
		}
	}

	chunkDir = config.ResolveDocumentsChunksDir(cfg, corpusID, chunkDir)
	contextDir = config.ResolveDocumentsContextsDir(cfg, corpusID, contextDir)
	embeddingDir = config.ResolveDocumentsEmbeddingsDir(cfg, corpusID, embeddingDir)
	embedEndpoint = config.ResolveEmbedEndpoint(cfg, embedEndpoint)
	embedAPIKey := config.ResolveEmbedAPIKey(cfg)
	embedAPI = config.ResolveEmbedAPI(cfg, embedAPI)
	model = config.ResolveEmbedModel(cfg, model)

	if documentID == "" {
		log.Fatal("document-id is required")
	}

	if err := run(log, documentID, chunkDir, contextDir, embeddingDir, embedEndpoint, embedAPIKey, embedAPI, model, modelVersion); err != nil {
		log.Fatal("Failed: %v", err)
	}
}

func run(log *logging.Logger, documentID, chunkDir, contextDir, embeddingDir, embedEndpoint, embedAPIKey, embedAPI, model, modelVersion string) error {
	chunkPath := filepath.Join(chunkDir, fmt.Sprintf("%s.jsonl", documentID))
	embeddingPath := filepath.Join(embeddingDir, fmt.Sprintf("%s.jsonl", documentID))

	if storage.MarkerExists(embeddingPath) {
		log.Info("Embeddings already exist: %s", embeddingPath)
		return nil
	}

	log.Info("Loading chunks: %s", chunkPath)

	var chunks []schema.TextChunk
	if err := storage.ReadJSONL(chunkPath, func(line []byte) error {
		var chunk schema.TextChunk
		if err := json.Unmarshal(line, &chunk); err != nil {
			return err
		}
		chunks = append(chunks, chunk)
		return nil
	}); err != nil {
		return fmt.Errorf("read chunks: %w", err)
	}

	contexts, err := contextual.LoadContexts(contextDir, documentID)
	if err != nil {
		return fmt.Errorf("read contexts: %w", err)
	}

	log.Info("Generating embeddings for %d chunks (%d with context)", len(chunks), len(contexts))

	if err := os.Remove(embeddingPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove old embeddings: %w", err)
	}

	var client *embed.Client
	if embedEndpoint != "" {
		client, err = embed.NewClient(embedEndpoint, embedAPIKey, embedAPI, model)
		if err != nil {
			return err
		}
	}

	for i, chunk := range chunks {
		var vector []float32
		var err error
		chunkContext := contextual.Lookup(contexts, chunk.ChunkID, chunk.TextHash)

		if client != nil {
			vector, err = client.Embed(contextual.EmbeddingText(chunkContext, chunk.Text))
			if err != nil {
				return fmt.Errorf("generate embedding for chunk %d: %w", i, err)
			}
		} else {
			log.Warn("No embedding endpoint configured, generating placeholder")
			vector = generatePlaceholderEmbedding()
		}

		embedding := schema.TextEmbedding{
			ChunkID:      chunk.ChunkID,
			DocumentID:   chunk.DocumentID,
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

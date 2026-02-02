package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"path/filepath"
	"slices"
	"time"

	"github.com/google/uuid"
	"github.com/jeffrpowell/raggo/pkg/config"
	"github.com/jeffrpowell/raggo/pkg/logging"
	"github.com/jeffrpowell/raggo/pkg/schema"
	"github.com/jeffrpowell/raggo/pkg/storage"
	qdrant "github.com/qdrant/go-client/qdrant"
)

func main() {
	var (
		configPath   string
		corpusID     string
		episodeID    string
		chunkDir     string
		embeddingDir string
		indexDir     string
		qdrantHost   string
		qdrantPort   int
		collection   string
		corpus       string
		sourceType   string
	)

	flag.StringVar(&configPath, "config", "", "Path to raggo.yml config file")
	flag.StringVar(&corpusID, "corpus-id", "", "Corpus ID from config")
	flag.StringVar(&episodeID, "episode-id", "", "Episode ID")
	flag.StringVar(&chunkDir, "chunk-dir", "data/chunks", "Directory for chunks")
	flag.StringVar(&embeddingDir, "embedding-dir", "data/embeddings", "Directory for embeddings")
	flag.StringVar(&indexDir, "index-dir", "data/index", "Directory for index markers")
	flag.StringVar(&qdrantHost, "qdrant-host", "localhost", "Qdrant host")
	flag.IntVar(&qdrantPort, "qdrant-port", 6334, "Qdrant gRPC port")
	flag.StringVar(&collection, "collection", "raggo", "Qdrant collection name")
	flag.StringVar(&corpus, "corpus", "default", "Corpus name")
	flag.StringVar(&sourceType, "source-type", "podcast", "Source type")
	flag.Parse()

	log := logging.New("raggo-index-podcast")

	var cfg *config.Config
	if configPath != "" {
		var err error
		cfg, err = config.Load(configPath)
		if err != nil {
			log.Fatal("Failed to load config: %v", err)
		}
	}

	chunkDir = config.ResolveChunksDir(cfg, corpusID, chunkDir)
	embeddingDir = config.ResolveEmbeddingsDir(cfg, corpusID, embeddingDir)
	indexDir = config.ResolveIndexDir(cfg, corpusID, indexDir)
	qdrantHost = config.ResolveQdrantHost(cfg, qdrantHost)
	qdrantPort = config.ResolveQdrantPort(cfg, qdrantPort)

	if episodeID == "" {
		log.Fatal("episode-id is required")
	}

	if err := run(log, episodeID, chunkDir, embeddingDir, indexDir, qdrantHost, qdrantPort, collection, corpus, sourceType); err != nil {
		log.Fatal("Failed: %v", err)
	}
}

func run(log *logging.Logger, episodeID, chunkDir, embeddingDir, indexDir, qdrantHost string, qdrantPort int, collection, corpus, sourceType string) error {
	markerPath := filepath.Join(indexDir, fmt.Sprintf("%s.done", episodeID))

	if storage.MarkerExists(markerPath) {
		log.Info("Episode already indexed: %s", episodeID)
		return nil
	}

	chunkPath := filepath.Join(chunkDir, fmt.Sprintf("%s.jsonl", episodeID))
	embeddingPath := filepath.Join(embeddingDir, fmt.Sprintf("%s.jsonl", episodeID))

	log.Info("Loading chunks and embeddings")

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

	var embeddings []schema.Embedding
	if err := storage.ReadJSONL(embeddingPath, func(line []byte) error {
		var emb schema.Embedding
		if err := json.Unmarshal(line, &emb); err != nil {
			return err
		}
		embeddings = append(embeddings, emb)
		return nil
	}); err != nil {
		return fmt.Errorf("read embeddings: %w", err)
	}

	if len(chunks) != len(embeddings) {
		return fmt.Errorf("chunk/embedding count mismatch: %d != %d", len(chunks), len(embeddings))
	}

	log.Info("Indexing %d chunks into Qdrant", len(chunks))

	client, err := qdrant.NewClient(&qdrant.Config{
		Host: qdrantHost,
		Port: qdrantPort,
	})
	if err != nil {
		return fmt.Errorf("connect to qdrant: %w", err)
	}
	defer client.Close()

	ctx := context.Background()

	exists, err := collectionExists(ctx, client, collection)
	if err != nil {
		return fmt.Errorf("check collection: %w", err)
	}

	if !exists {
		log.Info("Creating collection: %s", collection)
		if err := createCollection(ctx, client, collection, uint64(embeddings[0].Dimension)); err != nil {
			return fmt.Errorf("create collection: %w", err)
		}
	}

	for i := range chunks {
		chunk := &chunks[i]
		embedding := &embeddings[i]

		if chunk.ChunkID != embedding.ChunkID {
			return fmt.Errorf("chunk/embedding ID mismatch at index %d", i)
		}

		pointID := uuid.New().String()

		payload := map[string]interface{}{
			"point_id":      pointID,
			"chunk_id":      chunk.ChunkID,
			"episode_id":    episodeID,
			"corpus":        corpus,
			"source_type":   sourceType,
			"source_id":     episodeID,
			"start_time":    chunk.StartTime,
			"end_time":      chunk.EndTime,
			"text":          chunk.Text,
			"model":         embedding.Model,
			"model_version": embedding.ModelVersion,
			"indexed_at":    time.Now().Format(time.RFC3339),
		}

		point := &qdrant.PointStruct{
			Id:      qdrant.NewID(pointID),
			Vectors: qdrant.NewVectors(embedding.Vector...),
			Payload: qdrant.NewValueMap(payload),
		}

		if _, err := client.Upsert(ctx, &qdrant.UpsertPoints{
			CollectionName: collection,
			Points:         []*qdrant.PointStruct{point},
		}); err != nil {
			return fmt.Errorf("upsert point %d: %w", i, err)
		}

		if (i+1)%10 == 0 {
			log.Info("Indexed %d/%d chunks", i+1, len(chunks))
		}
	}

	if err := storage.WriteMarker(markerPath); err != nil {
		return fmt.Errorf("write marker: %w", err)
	}

	log.Info("Successfully indexed episode: %s", episodeID)
	return nil
}

func collectionExists(ctx context.Context, client *qdrant.Client, name string) (bool, error) {
	collections, err := client.ListCollections(ctx)
	if err != nil {
		return false, err
	}

	return slices.Contains(collections, name), nil
}

func createCollection(ctx context.Context, client *qdrant.Client, name string, dimension uint64) error {
	return client.CreateCollection(ctx, &qdrant.CreateCollection{
		CollectionName: name,
		VectorsConfig: qdrant.NewVectorsConfig(&qdrant.VectorParams{
			Size:     dimension,
			Distance: qdrant.Distance_Cosine,
		}),
	})
}

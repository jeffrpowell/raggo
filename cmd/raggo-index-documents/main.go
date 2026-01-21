package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	"github.com/jeffrpowell/raggo/pkg/logging"
	"github.com/jeffrpowell/raggo/pkg/schema"
	"github.com/jeffrpowell/raggo/pkg/storage"
	"github.com/qdrant/go-client/qdrant"
)

func main() {
	var (
		documentID     string
		manifestDir    string
		chunkDir       string
		embeddingDir   string
		indexDir       string
		qdrantHost     string
		qdrantPort     int
		collection     string
		corpus         string
		sourceType     string
	)

	flag.StringVar(&documentID, "document-id", "", "Document ID")
	flag.StringVar(&manifestDir, "manifest-dir", "data/manifests", "Directory for manifests")
	flag.StringVar(&chunkDir, "chunk-dir", "data/documents/chunks", "Directory for chunks")
	flag.StringVar(&embeddingDir, "embedding-dir", "data/documents/embeddings", "Directory for embeddings")
	flag.StringVar(&indexDir, "index-dir", "data/documents/index", "Directory for index markers")
	flag.StringVar(&qdrantHost, "qdrant-host", "localhost", "Qdrant host")
	flag.IntVar(&qdrantPort, "qdrant-port", 6334, "Qdrant gRPC port")
	flag.StringVar(&collection, "collection", "raggo", "Qdrant collection name")
	flag.StringVar(&corpus, "corpus", "default", "Corpus name")
	flag.StringVar(&sourceType, "source-type", "document", "Source type")
	flag.Parse()

	log := logging.New("raggo-index-documents")

	if documentID == "" {
		log.Fatal("document-id is required")
	}

	if err := run(log, documentID, manifestDir, chunkDir, embeddingDir, indexDir, qdrantHost, qdrantPort, collection, corpus, sourceType); err != nil {
		log.Fatal("Failed: %v", err)
	}
}

func run(log *logging.Logger, documentID, manifestDir, chunkDir, embeddingDir, indexDir, qdrantHost string, qdrantPort int, collection, corpus, sourceType string) error {
	markerPath := filepath.Join(indexDir, fmt.Sprintf("%s.done", documentID))

	if storage.MarkerExists(markerPath) {
		log.Info("Document already indexed: %s", documentID)
		return nil
	}

	chunkPath := filepath.Join(chunkDir, fmt.Sprintf("%s.jsonl", documentID))
	embeddingPath := filepath.Join(embeddingDir, fmt.Sprintf("%s.jsonl", documentID))

	log.Info("Loading chunks and embeddings")

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

	var embeddings []schema.TextEmbedding
	if err := storage.ReadJSONL(embeddingPath, func(line []byte) error {
		var emb schema.TextEmbedding
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

	var manifest *schema.DocumentManifest
	if err := findDocumentManifest(manifestDir, documentID, &manifest); err != nil {
		log.Warn("Could not load document manifest: %v", err)
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
			"document_id":   documentID,
			"corpus":        corpus,
			"source_type":   sourceType,
			"text":          chunk.Text,
			"model":         embedding.Model,
			"model_version": embedding.ModelVersion,
			"indexed_at":    time.Now().Format(time.RFC3339),
		}

		if manifest != nil {
			payload["source_path"] = manifest.RelativePath
			payload["file_name"] = manifest.FileName
			payload["file_type"] = manifest.FileType
		}

		point := &qdrant.PointStruct{
			Id:      qdrant.NewIDString(pointID),
			Vectors: qdrant.NewVectors(embedding.Vector...),
			Payload: qdrant.NewValueMap(payload),
		}

		if err := client.Upsert(ctx, &qdrant.UpsertPoints{
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

	log.Info("Successfully indexed document: %s", documentID)
	return nil
}

func findDocumentManifest(manifestDir, documentID string, result **schema.DocumentManifest) error {
	files, err := filepath.Glob(filepath.Join(manifestDir, "*.jsonl"))
	if err != nil {
		return err
	}

	for _, file := range files {
		if err := storage.ReadJSONL(file, func(line []byte) error {
			var manifest schema.DocumentManifest
			if err := json.Unmarshal(line, &manifest); err != nil {
				return err
			}
			if manifest.DocumentID == documentID {
				*result = &manifest
				return fmt.Errorf("found")
			}
			return nil
		}); err != nil {
			if err.Error() == "found" {
				return nil
			}
			return err
		}
	}

	return fmt.Errorf("manifest not found")
}

func collectionExists(ctx context.Context, client *qdrant.Client, name string) (bool, error) {
	collections, err := client.ListCollections(ctx)
	if err != nil {
		return false, err
	}

	for _, coll := range collections {
		if coll.Name == name {
			return true, nil
		}
	}

	return false, nil
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

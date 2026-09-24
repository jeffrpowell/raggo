// Package vectorstore defines the Qdrant collection layout shared by the
// document and podcast index stages: a named dense vector plus a sparse BM25
// vector computed server-side by Qdrant (requires Qdrant >= 1.15.2).
package vectorstore

import (
	"context"
	"fmt"
	"slices"

	qdrant "github.com/qdrant/go-client/qdrant"
)

const (
	DenseVector  = "dense"
	SparseVector = "bm25"
	BM25Model    = "qdrant/bm25"
)

// EnsureHybridCollection creates the collection if missing, or verifies an
// existing one already uses the dense + bm25 layout.
func EnsureHybridCollection(ctx context.Context, client *qdrant.Client, name string, dimension uint64) (created bool, err error) {
	collections, err := client.ListCollections(ctx)
	if err != nil {
		return false, fmt.Errorf("list collections: %w", err)
	}

	if !slices.Contains(collections, name) {
		err := client.CreateCollection(ctx, &qdrant.CreateCollection{
			CollectionName: name,
			VectorsConfig: qdrant.NewVectorsConfigMap(map[string]*qdrant.VectorParams{
				DenseVector: {
					Size:     dimension,
					Distance: qdrant.Distance_Cosine,
				},
			}),
			SparseVectorsConfig: qdrant.NewSparseVectorsConfig(map[string]*qdrant.SparseVectorParams{
				SparseVector: {
					Modifier: qdrant.Modifier_Idf.Enum(),
				},
			}),
		})
		if err != nil {
			return false, fmt.Errorf("create collection: %w", err)
		}
		return true, nil
	}

	info, err := client.GetCollectionInfo(ctx, name)
	if err != nil {
		return false, fmt.Errorf("get collection info: %w", err)
	}
	params := info.GetConfig().GetParams()
	dense := params.GetVectorsConfig().GetParamsMap().GetMap()[DenseVector]
	sparse := params.GetSparseVectorsConfig().GetMap()[SparseVector]
	if dense == nil || sparse == nil {
		return false, fmt.Errorf("collection %s uses a legacy schema (expected named vectors %q and sparse %q); delete it or pick a new collection name and re-index", name, DenseVector, SparseVector)
	}
	if dense.GetSize() != dimension {
		return false, fmt.Errorf("collection %s has dense dimension %d, embeddings have %d", name, dense.GetSize(), dimension)
	}
	return false, nil
}

// HybridVectors builds the point vectors: the precomputed dense embedding and
// a BM25 document that Qdrant tokenizes server-side.
func HybridVectors(dense []float32, bm25Text string) *qdrant.Vectors {
	return qdrant.NewVectorsMap(map[string]*qdrant.Vector{
		DenseVector: qdrant.NewVectorDense(dense),
		SparseVector: qdrant.NewVectorDocument(&qdrant.Document{
			Text:  bm25Text,
			Model: BM25Model,
		}),
	})
}

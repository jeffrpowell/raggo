package state

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"

	"github.com/jeffrpowell/raggo/pkg/config"
	qdrant "github.com/qdrant/go-client/qdrant"
)

type QdrantStateManager struct {
	client     *qdrant.Client
	collection string
	ctx        context.Context
}

func NewQdrantStateManager(cfg config.QdrantConfig) (*QdrantStateManager, error) {
	client, err := qdrant.NewClient(&qdrant.Config{
		Host: cfg.Host,
		Port: cfg.Port,
	})
	if err != nil {
		return nil, fmt.Errorf("connect to qdrant: %w", err)
	}

	sm := &QdrantStateManager{
		client:     client,
		collection: cfg.MetadataCollection,
		ctx:        context.Background(),
	}

	if err := sm.ensureCollection(); err != nil {
		return nil, fmt.Errorf("ensure collection: %w", err)
	}

	return sm, nil
}

func (sm *QdrantStateManager) Close() error {
	return sm.client.Close()
}

func (sm *QdrantStateManager) ensureCollection() error {
	exists, err := sm.collectionExists()
	if err != nil {
		return err
	}

	if !exists {
		err = sm.client.CreateCollection(sm.ctx, &qdrant.CreateCollection{
			CollectionName: sm.collection,
			VectorsConfig: qdrant.NewVectorsConfig(&qdrant.VectorParams{
				Size:     1,
				Distance: qdrant.Distance_Cosine,
			}),
		})
		if err != nil {
			return fmt.Errorf("create collection: %w", err)
		}
	}

	return nil
}

func (sm *QdrantStateManager) collectionExists() (bool, error) {
	collections, err := sm.client.ListCollections(sm.ctx)
	if err != nil {
		return false, err
	}
	return slices.Contains(collections, sm.collection), nil
}

func (sm *QdrantStateManager) storeMetadata(pointID string, metadata interface{}) error {
	jsonData, err := json.Marshal(metadata)
	if err != nil {
		return fmt.Errorf("marshal metadata: %w", err)
	}

	var payloadMap map[string]interface{}
	if err := json.Unmarshal(jsonData, &payloadMap); err != nil {
		return fmt.Errorf("unmarshal to map: %w", err)
	}

	point := &qdrant.PointStruct{
		Id:      qdrant.NewID(pointID),
		Vectors: qdrant.NewVectors(0.0),
		Payload: qdrant.NewValueMap(payloadMap),
	}

	_, err = sm.client.Upsert(sm.ctx, &qdrant.UpsertPoints{
		CollectionName: sm.collection,
		Points:         []*qdrant.PointStruct{point},
	})

	return err
}

func (sm *QdrantStateManager) searchByField(field, value string) ([]*qdrant.ScoredPoint, error) {
	filter := &qdrant.Filter{
		Must: []*qdrant.Condition{
			{
				ConditionOneOf: &qdrant.Condition_Field{
					Field: &qdrant.FieldCondition{
						Key: field,
						Match: &qdrant.Match{
							MatchValue: &qdrant.Match_Keyword{
								Keyword: value,
							},
						},
					},
				},
			},
		},
	}

	result, err := sm.client.Query(sm.ctx, &qdrant.QueryPoints{
		CollectionName: sm.collection,
		Query:          qdrant.NewQuery(0.0),
		Filter:         filter,
		Limit:          uintPtr(100),
		WithPayload:    qdrant.NewWithPayload(true),
	})

	if err != nil {
		return nil, err
	}

	return result, nil
}

func uintPtr(v uint64) *uint64 {
	return &v
}

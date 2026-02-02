package state

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	qdrant "github.com/qdrant/go-client/qdrant"
)

func (sm *QdrantStateManager) StartRun(corpusID string) (string, error) {
	runID := uuid.New().String()

	metadata := CorpusRunMetadata{
		CorpusID:       corpusID,
		RunID:          runID,
		StartedAt:      time.Now(),
		Status:         "running",
		ItemsFound:     0,
		ItemsProcessed: 0,
		ItemsIndexed:   0,
		Stage:          "init",
	}

	if err := sm.storeMetadata(runID, metadata); err != nil {
		return "", fmt.Errorf("store run metadata: %w", err)
	}

	return runID, nil
}

func (sm *QdrantStateManager) UpdateRunProgress(runID string, stage string, itemsProcessed int) error {
	points, err := sm.searchByField("run_id", runID)
	if err != nil {
		return fmt.Errorf("search for run: %w", err)
	}

	if len(points) == 0 {
		return fmt.Errorf("run not found: %s", runID)
	}

	point := points[0]
	payload := point.Payload

	payload["stage"] = &qdrant.Value{Kind: &qdrant.Value_StringValue{StringValue: stage}}
	payload["items_processed"] = &qdrant.Value{Kind: &qdrant.Value_IntegerValue{IntegerValue: int64(itemsProcessed)}}

	updatedPoint := &qdrant.PointStruct{
		Id:      point.Id,
		Vectors: qdrant.NewVectors(0.0),
		Payload: payload,
	}

	_, err = sm.client.Upsert(sm.ctx, &qdrant.UpsertPoints{
		CollectionName: sm.collection,
		Points:         []*qdrant.PointStruct{updatedPoint},
	})

	return err
}

func (sm *QdrantStateManager) CompleteRun(runID string, status string, errorMsg string) error {
	points, err := sm.searchByField("run_id", runID)
	if err != nil {
		return fmt.Errorf("search for run: %w", err)
	}

	if len(points) == 0 {
		return fmt.Errorf("run not found: %s", runID)
	}

	point := points[0]
	payload := point.Payload

	payload["status"] = &qdrant.Value{Kind: &qdrant.Value_StringValue{StringValue: status}}
	payload["completed_at"] = &qdrant.Value{Kind: &qdrant.Value_StringValue{StringValue: time.Now().Format(time.RFC3339)}}
	if errorMsg != "" {
		payload["error_message"] = &qdrant.Value{Kind: &qdrant.Value_StringValue{StringValue: errorMsg}}
	}

	updatedPoint := &qdrant.PointStruct{
		Id:      point.Id,
		Vectors: qdrant.NewVectors(0.0),
		Payload: payload,
	}

	_, err = sm.client.Upsert(sm.ctx, &qdrant.UpsertPoints{
		CollectionName: sm.collection,
		Points:         []*qdrant.PointStruct{updatedPoint},
	})

	return err
}

func (sm *QdrantStateManager) GetLastRun(corpusID string) (*CorpusRunMetadata, error) {
	points, err := sm.searchByField("corpus_id", corpusID)
	if err != nil {
		return nil, fmt.Errorf("search for corpus runs: %w", err)
	}

	if len(points) == 0 {
		return nil, nil
	}

	var latestRun *CorpusRunMetadata
	var latestTime time.Time

	for _, point := range points {
		jsonData, err := json.Marshal(point.Payload)
		if err != nil {
			continue
		}

		var payloadMap map[string]interface{}
		if err := json.Unmarshal(jsonData, &payloadMap); err != nil {
			continue
		}

		var run CorpusRunMetadata
		jsonData, _ = json.Marshal(payloadMap)
		if err := json.Unmarshal(jsonData, &run); err != nil {
			continue
		}

		if run.StartedAt.After(latestTime) {
			latestTime = run.StartedAt
			latestRun = &run
		}
	}

	return latestRun, nil
}

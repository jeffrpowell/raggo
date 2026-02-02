package state

import (
	"encoding/json"
	"fmt"
)

func (sm *QdrantStateManager) GetWatchState(corpusID string) (*WatchState, error) {
	points, err := sm.searchByField("corpus_id", corpusID)
	if err != nil {
		return nil, fmt.Errorf("search for watch state: %w", err)
	}

	for _, point := range points {
		jsonData, err := json.Marshal(point.Payload)
		if err != nil {
			continue
		}

		var payloadMap map[string]interface{}
		if err := json.Unmarshal(jsonData, &payloadMap); err != nil {
			continue
		}

		if _, ok := payloadMap["content_hash"]; ok {
			var watchState WatchState
			jsonData, _ = json.Marshal(payloadMap)
			if err := json.Unmarshal(jsonData, &watchState); err != nil {
				continue
			}
			return &watchState, nil
		}
	}

	return nil, nil
}

func (sm *QdrantStateManager) UpdateWatchState(corpusID string, state *WatchState) error {
	pointID := fmt.Sprintf("watch-%s", corpusID)
	return sm.storeMetadata(pointID, state)
}

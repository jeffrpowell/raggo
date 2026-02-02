package state

import "time"

type CorpusRunMetadata struct {
	CorpusID       string    `json:"corpus_id"`
	RunID          string    `json:"run_id"`
	StartedAt      time.Time `json:"started_at"`
	CompletedAt    time.Time `json:"completed_at,omitempty"`
	Status         string    `json:"status"`
	ErrorMessage   string    `json:"error_message,omitempty"`
	ItemsFound     int       `json:"items_found"`
	ItemsProcessed int       `json:"items_processed"`
	ItemsIndexed   int       `json:"items_indexed"`
	Stage          string    `json:"stage"`
}

type WatchState struct {
	CorpusID      string            `json:"corpus_id"`
	LastCheckedAt time.Time         `json:"last_checked_at"`
	ContentHash   string            `json:"content_hash"`
	ItemHashes    map[string]string `json:"item_hashes"`
}

type StateManager interface {
	StartRun(corpusID string) (runID string, err error)
	UpdateRunProgress(runID string, stage string, itemsProcessed int) error
	CompleteRun(runID string, status string, errorMsg string) error
	GetLastRun(corpusID string) (*CorpusRunMetadata, error)
	
	GetWatchState(corpusID string) (*WatchState, error)
	UpdateWatchState(corpusID string, state *WatchState) error
}

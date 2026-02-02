package main

import (
	"crypto/sha256"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/jeffrpowell/raggo/pkg/config"
	"github.com/jeffrpowell/raggo/pkg/logging"
	"github.com/jeffrpowell/raggo/pkg/pipeline"
	"github.com/jeffrpowell/raggo/pkg/state"
)

func runWatchMode(executor *pipeline.Executor, corpora []config.CorpusConfig, logger *logging.Logger, stateManager state.StateManager) {
	logger.Info("Starting watch mode with %d corpora", len(corpora))

	for _, corpus := range corpora {
		corpus := corpus
		go func() {
			if corpus.Type == "podcast" {
				watchPodcastCorpus(executor, corpus, stateManager, logger)
			} else if corpus.Type == "documents" {
				watchDocumentCorpus(executor, corpus, stateManager, logger)
			}
		}()
	}

	select {}
}

func watchPodcastCorpus(executor *pipeline.Executor, corpus config.CorpusConfig, stateManager state.StateManager, logger *logging.Logger) {
	interval := time.Duration(corpus.WatchIntervalSeconds) * time.Second
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for range ticker.C {
		logger.Debug("Checking RSS feed for corpus: %s", corpus.ID)

		feedURL := corpus.Config["feed_url"].(string)
		currentHash, err := fetchRSSHash(feedURL)
		if err != nil {
			logger.Error("Failed to fetch RSS for corpus: %s, error: %v", corpus.ID, err)
			continue
		}

		watchState, err := stateManager.GetWatchState(corpus.ID)
		if err != nil {
			logger.Error("Failed to get watch state for corpus: %s, error: %v", corpus.ID, err)
			continue
		}

		if watchState != nil && watchState.ContentHash == currentHash {
			logger.Debug("No changes detected for corpus: %s", corpus.ID)
			continue
		}

		logger.Info("RSS feed changed, triggering pipeline for corpus: %s", corpus.ID)
		if err := executor.ExecutePipelineWithRetry(corpus); err != nil {
			logger.Error("Watch-triggered run failed for corpus: %s, error: %v", corpus.ID, err)
			continue
		}

		stateManager.UpdateWatchState(corpus.ID, &state.WatchState{
			CorpusID:      corpus.ID,
			LastCheckedAt: time.Now(),
			ContentHash:   currentHash,
		})
	}
}

func watchDocumentCorpus(executor *pipeline.Executor, corpus config.CorpusConfig, stateManager state.StateManager, logger *logging.Logger) {
	interval := time.Duration(corpus.WatchIntervalSeconds) * time.Second
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for range ticker.C {
		logger.Debug("Checking document directory for corpus: %s", corpus.ID)

		rootPath := corpus.Config["root_path"].(string)
		currentHash, err := hashDirectoryTree(rootPath)
		if err != nil {
			logger.Error("Failed to hash directory for corpus: %s, error: %v", corpus.ID, err)
			continue
		}

		watchState, err := stateManager.GetWatchState(corpus.ID)
		if err != nil {
			logger.Error("Failed to get watch state for corpus: %s, error: %v", corpus.ID, err)
			continue
		}

		if watchState != nil && watchState.ContentHash == currentHash {
			logger.Debug("No changes detected for corpus: %s", corpus.ID)
			continue
		}

		logger.Info("Directory changed, triggering pipeline for corpus: %s", corpus.ID)
		if err := executor.ExecutePipelineWithRetry(corpus); err != nil {
			logger.Error("Watch-triggered run failed for corpus: %s, error: %v", corpus.ID, err)
			continue
		}

		stateManager.UpdateWatchState(corpus.ID, &state.WatchState{
			CorpusID:      corpus.ID,
			LastCheckedAt: time.Now(),
			ContentHash:   currentHash,
		})
	}
}

func fetchRSSHash(url string) (string, error) {
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("http status: %d", resp.StatusCode)
	}

	h := sha256.New()
	if _, err := io.Copy(h, resp.Body); err != nil {
		return "", err
	}

	return fmt.Sprintf("%x", h.Sum(nil)), nil
}

func hashDirectoryTree(rootPath string) (string, error) {
	h := sha256.New()

	err := filepath.Walk(rootPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}

		relPath, err := filepath.Rel(rootPath, path)
		if err != nil {
			return nil
		}

		h.Write([]byte(relPath))
		h.Write([]byte(fmt.Sprintf("%d", info.ModTime().Unix())))
		h.Write([]byte(fmt.Sprintf("%d", info.Size())))

		return nil
	})

	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%x", h.Sum(nil)), nil
}

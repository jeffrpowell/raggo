package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/jeffrpowell/raggo/pkg/concurrency"
	"github.com/jeffrpowell/raggo/pkg/hashing"
	"github.com/jeffrpowell/raggo/pkg/logging"
	"github.com/jeffrpowell/raggo/pkg/schema"
	"github.com/jeffrpowell/raggo/pkg/storage"
)

func main() {
	var (
		manifestPath string
		audioDir     string
		workers      int
	)

	flag.StringVar(&manifestPath, "manifest", "", "Path to episode manifest JSONL file")
	flag.StringVar(&audioDir, "audio-dir", "data/audio", "Directory for audio files")
	flag.IntVar(&workers, "workers", 4, "Number of concurrent downloads")
	flag.Parse()

	log := logging.New("raggo-download-audio")

	if manifestPath == "" {
		log.Fatal("manifest is required")
	}

	if err := run(log, manifestPath, audioDir, workers); err != nil {
		log.Fatal("Failed: %v", err)
	}
}

func run(log *logging.Logger, manifestPath, audioDir string, workers int) error {
	var episodes []schema.EpisodeManifest

	if err := storage.ReadJSONL(manifestPath, func(line []byte) error {
		var ep schema.EpisodeManifest
		if err := json.Unmarshal(line, &ep); err != nil {
			return err
		}
		episodes = append(episodes, ep)
		return nil
	}); err != nil {
		return fmt.Errorf("read manifest: %w", err)
	}

	log.Info("Found %d episodes to process", len(episodes))

	pool := concurrency.NewWorkerPool(workers)

	for _, ep := range episodes {
		episode := ep
		pool.Submit(func() error {
			return downloadEpisode(log, episode, audioDir)
		})
	}

	if err := pool.Wait(); err != nil {
		return fmt.Errorf("download failed: %w", err)
	}

	log.Info("Successfully downloaded all episodes")
	return nil
}

func downloadEpisode(log *logging.Logger, ep schema.EpisodeManifest, audioDir string) error {
	ext := getExtension(ep.AudioURL)
	audioPath := filepath.Join(audioDir, fmt.Sprintf("%s%s", ep.EpisodeID, ext))
	metaPath := filepath.Join(audioDir, fmt.Sprintf("%s.json", ep.EpisodeID))

	if storage.MarkerExists(audioPath) && storage.MarkerExists(metaPath) {
		log.Info("Skipping existing: %s", ep.Title)
		return nil
	}

	log.Info("Downloading: %s", ep.Title)

	resp, err := http.Get(ep.AudioURL)
	if err != nil {
		return fmt.Errorf("http get: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("http status: %d", resp.StatusCode)
	}

	if err := storage.EnsureDir(audioDir); err != nil {
		return err
	}

	tmpPath := audioPath + ".tmp"
	f, err := os.Create(tmpPath)
	if err != nil {
		return err
	}

	written, err := io.Copy(f, resp.Body)
	if closeErr := f.Close(); closeErr != nil && err == nil {
		err = closeErr
	}
	if err != nil {
		if removeErr := os.Remove(tmpPath); removeErr != nil && !os.IsNotExist(removeErr) {
			return fmt.Errorf("copy failed: %w, cleanup failed: %v", err, removeErr)
		}
		return err
	}

	if err := os.Rename(tmpPath, audioPath); err != nil {
		return err
	}

	contentHash, err := hashing.HashFile(audioPath)
	if err != nil {
		return err
	}

	metadata := schema.AudioMetadata{
		EpisodeID:    ep.EpisodeID,
		FilePath:     audioPath,
		FileSize:     written,
		ContentHash:  contentHash,
		MimeType:     resp.Header.Get("Content-Type"),
		Duration:     0,
		DownloadedAt: time.Now(),
	}

	if err := storage.WriteJSON(metaPath, metadata); err != nil {
		return err
	}

	log.Info("Downloaded: %s (%d bytes)", ep.Title, written)
	return nil
}

func getExtension(url string) string {
	ext := filepath.Ext(url)
	if ext == "" || len(ext) > 5 {
		return ".mp3"
	}
	return ext
}

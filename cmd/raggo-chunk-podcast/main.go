package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/jeffrpowell/raggo/pkg/config"
	"github.com/jeffrpowell/raggo/pkg/hashing"
	"github.com/jeffrpowell/raggo/pkg/logging"
	"github.com/jeffrpowell/raggo/pkg/schema"
	"github.com/jeffrpowell/raggo/pkg/storage"
)

func main() {
	var (
		configPath     string
		corpusID       string
		episodeID      string
		transcriptDir  string
		chunkDir       string
		windowSeconds  float64
		overlapSeconds float64
	)

	flag.StringVar(&configPath, "config", "", "Path to raggo.yml config file")
	flag.StringVar(&corpusID, "corpus-id", "", "Corpus ID from config")
	flag.StringVar(&episodeID, "episode-id", "", "Episode ID")
	flag.StringVar(&transcriptDir, "transcript-dir", "data/transcripts", "Directory for transcripts")
	flag.StringVar(&chunkDir, "chunk-dir", "data/chunks", "Directory for chunks")
	flag.Float64Var(&windowSeconds, "window", 60.0, "Chunk window size in seconds")
	flag.Float64Var(&overlapSeconds, "overlap", 5.0, "Overlap size in seconds")
	flag.Parse()

	log := logging.New("raggo-chunk-podcast")

	var cfg *config.Config
	if configPath != "" {
		var err error
		cfg, err = config.Load(configPath)
		if err != nil {
			log.Fatal("Failed to load config: %v", err)
		}
	}

	transcriptDir = config.ResolveTranscriptsDir(cfg, corpusID, transcriptDir)
	chunkDir = config.ResolveChunksDir(cfg, corpusID, chunkDir)

	if episodeID == "" {
		log.Fatal("episode-id is required")
	}

	if err := run(log, episodeID, transcriptDir, chunkDir, windowSeconds, overlapSeconds); err != nil {
		log.Fatal("Failed: %v", err)
	}
}

func run(log *logging.Logger, episodeID, transcriptDir, chunkDir string, windowSeconds, overlapSeconds float64) error {
	normalizedPath := filepath.Join(transcriptDir, fmt.Sprintf("%s.normalized.json", episodeID))
	chunkPath := filepath.Join(chunkDir, fmt.Sprintf("%s.jsonl", episodeID))

	if storage.MarkerExists(chunkPath) {
		log.Info("Chunks already exist: %s", chunkPath)
		return nil
	}

	log.Info("Loading normalized transcript: %s", normalizedPath)

	var transcript schema.NormalizedTranscript
	if err := storage.ReadJSON(normalizedPath, &transcript); err != nil {
		return fmt.Errorf("read normalized transcript: %w", err)
	}

	log.Info("Chunking %d segments (window=%.1fs, overlap=%.1fs)", 
		len(transcript.Segments), windowSeconds, overlapSeconds)

	chunks := createChunks(episodeID, transcript.Segments, windowSeconds, overlapSeconds)

	if err := os.Remove(chunkPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove old chunks: %w", err)
	}

	for _, chunk := range chunks {
		if err := storage.AppendJSONL(chunkPath, chunk); err != nil {
			return fmt.Errorf("append chunk: %w", err)
		}
	}

	log.Info("Wrote %d chunks to: %s", len(chunks), chunkPath)
	return nil
}

func createChunks(episodeID string, segments []schema.TranscriptSegment, windowSize, overlap float64) []schema.Chunk {
	if len(segments) == 0 {
		return nil
	}

	var chunks []schema.Chunk
	chunkIndex := 0
	
	startTime := 0.0
	totalDuration := segments[len(segments)-1].End

	for startTime < totalDuration {
		endTime := startTime + windowSize
		
		var chunkSegments []schema.TranscriptSegment
		for _, seg := range segments {
			if seg.End <= startTime {
				continue
			}
			if seg.Start >= endTime {
				break
			}
			chunkSegments = append(chunkSegments, seg)
		}

		if len(chunkSegments) == 0 {
			break
		}

		text := buildChunkText(chunkSegments)
		textHash := hashing.HashString(text)

		chunkID := hashing.HashString(fmt.Sprintf("%s|%d", episodeID, chunkIndex))

		chunk := schema.Chunk{
			ChunkID:    chunkID,
			EpisodeID:  episodeID,
			ChunkIndex: chunkIndex,
			StartTime:  startTime,
			EndTime:    endTime,
			Text:       text,
			TextHash:   textHash,
			CreatedAt:  time.Now(),
		}

		if overlap > 0 {
			chunk.OverlapStart = startTime
			chunk.OverlapEnd = startTime + overlap
		}

		chunks = append(chunks, chunk)
		chunkIndex++
		
		startTime += windowSize - overlap
	}

	return chunks
}

func buildChunkText(segments []schema.TranscriptSegment) string {
	var text string
	for i, seg := range segments {
		if i > 0 {
			text += " "
		}
		text += seg.Text
	}
	return text
}

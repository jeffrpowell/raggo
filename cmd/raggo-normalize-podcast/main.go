package main

import (
	"flag"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/jeffrpowell/raggo/pkg/hashing"
	"github.com/jeffrpowell/raggo/pkg/logging"
	"github.com/jeffrpowell/raggo/pkg/schema"
	"github.com/jeffrpowell/raggo/pkg/storage"
)

var (
	whitespaceRe = regexp.MustCompile(`\s+`)
	punctuationRe = regexp.MustCompile(`([.!?])\s*([A-Z])`)
)

func main() {
	var (
		episodeID     string
		transcriptDir string
	)

	flag.StringVar(&episodeID, "episode-id", "", "Episode ID")
	flag.StringVar(&transcriptDir, "transcript-dir", "data/transcripts", "Directory for transcripts")
	flag.Parse()

	log := logging.New("raggo-normalize-podcast")

	if episodeID == "" {
		log.Fatal("episode-id is required")
	}

	if err := run(log, episodeID, transcriptDir); err != nil {
		log.Fatal("Failed: %v", err)
	}
}

func run(log *logging.Logger, episodeID, transcriptDir string) error {
	rawPath := filepath.Join(transcriptDir, fmt.Sprintf("%s.json", episodeID))
	normalizedPath := filepath.Join(transcriptDir, fmt.Sprintf("%s.normalized.json", episodeID))

	if storage.MarkerExists(normalizedPath) {
		log.Info("Normalized transcript already exists: %s", normalizedPath)
		return nil
	}

	log.Info("Loading raw transcript: %s", rawPath)

	var rawTranscript schema.RawTranscript
	if err := storage.ReadJSON(rawPath, &rawTranscript); err != nil {
		return fmt.Errorf("read raw transcript: %w", err)
	}

	log.Info("Normalizing %d segments", len(rawTranscript.Segments))

	normalized := make([]schema.TranscriptSegment, len(rawTranscript.Segments))
	for i, seg := range rawTranscript.Segments {
		normalized[i] = normalizeSegment(seg)
	}

	result := schema.NormalizedTranscript{
		EpisodeID:         episodeID,
		AudioHash:         rawTranscript.AudioHash,
		RawTranscriptHash: rawTranscript.TranscriptHash,
		Segments:          normalized,
		NormalizedAt:      time.Now(),
	}

	normalizedHash, err := hashing.HashStruct(result)
	if err != nil {
		return fmt.Errorf("hash normalized: %w", err)
	}
	result.NormalizedHash = normalizedHash

	if err := storage.WriteJSON(normalizedPath, result); err != nil {
		return fmt.Errorf("write normalized: %w", err)
	}

	log.Info("Wrote normalized transcript: %s", normalizedPath)
	return nil
}

func normalizeSegment(seg schema.TranscriptSegment) schema.TranscriptSegment {
	text := seg.Text

	text = strings.TrimSpace(text)

	text = whitespaceRe.ReplaceAllString(text, " ")

	text = punctuationRe.ReplaceAllString(text, "$1 $2")

	speaker := normalizeSpeaker(seg.Speaker)

	return schema.TranscriptSegment{
		Start:   seg.Start,
		End:     seg.End,
		Text:    text,
		Speaker: speaker,
	}
}

func normalizeSpeaker(speaker string) string {
	speaker = strings.TrimSpace(speaker)
	if speaker == "" {
		return "Unknown"
	}
	
	speaker = strings.ToUpper(speaker[:1]) + strings.ToLower(speaker[1:])
	
	return speaker
}

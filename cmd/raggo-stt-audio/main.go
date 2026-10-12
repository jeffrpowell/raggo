package main

import (
	"flag"
	"fmt"
	"path/filepath"
	"time"

	"github.com/jeffrpowell/raggo/pkg/config"
	"github.com/jeffrpowell/raggo/pkg/hashing"
	"github.com/jeffrpowell/raggo/pkg/logging"
	"github.com/jeffrpowell/raggo/pkg/schema"
	"github.com/jeffrpowell/raggo/pkg/storage"
	"github.com/jeffrpowell/raggo/pkg/stt"
)

func main() {
	var (
		configPath    string
		corpusID      string
		episodeID     string
		audioPath     string
		transcriptDir string
		sttEndpoint   string
		sttAPI        string
		model         string
		modelVersion  string
	)

	flag.StringVar(&configPath, "config", "", "Path to raggo.yml config file")
	flag.StringVar(&corpusID, "corpus-id", "", "Corpus ID from config")
	flag.StringVar(&episodeID, "episode-id", "", "Episode ID")
	flag.StringVar(&audioPath, "audio-path", "", "Path to audio file")
	flag.StringVar(&transcriptDir, "transcript-dir", "data/transcripts", "Directory for transcripts")
	flag.StringVar(&sttEndpoint, "stt-endpoint", "", "STT service HTTP endpoint (optional)")
	flag.StringVar(&sttAPI, "stt-api", "", "STT request format: openai or raggo (default openai)")
	flag.StringVar(&model, "model", "", "STT model name (default whisper-1)")
	flag.StringVar(&modelVersion, "model-version", "v1", "STT model version")
	flag.Parse()

	log := logging.New("raggo-stt-audio")

	var cfg *config.Config
	if configPath != "" {
		var err error
		cfg, err = config.Load(configPath)
		if err != nil {
			log.Fatal("Failed to load config: %v", err)
		}
	}

	transcriptDir = config.ResolveTranscriptsDir(cfg, corpusID, transcriptDir)
	sttEndpoint = config.ResolveSTTEndpoint(cfg, sttEndpoint)
	sttAPIKey := config.ResolveSTTAPIKey(cfg)
	sttAPI = config.ResolveSTTAPI(cfg, sttAPI)
	model = config.ResolveSTTModel(cfg, model)

	if episodeID == "" || audioPath == "" {
		log.Fatal("episode-id and audio-path are required")
	}

	if err := run(log, episodeID, audioPath, transcriptDir, sttEndpoint, sttAPIKey, sttAPI, model, modelVersion); err != nil {
		log.Fatal("Failed: %v", err)
	}
}

func run(log *logging.Logger, episodeID, audioPath, transcriptDir, sttEndpoint, sttAPIKey, sttAPI, model, modelVersion string) error {
	transcriptPath := filepath.Join(transcriptDir, fmt.Sprintf("%s.json", episodeID))

	if storage.MarkerExists(transcriptPath) {
		log.Info("Transcript already exists: %s", transcriptPath)
		return nil
	}

	log.Info("Transcribing audio: %s", audioPath)

	audioHash, err := hashing.HashFile(audioPath)
	if err != nil {
		return fmt.Errorf("hash audio: %w", err)
	}

	var segments []schema.TranscriptSegment

	if sttEndpoint != "" {
		client, err := stt.NewClient(sttEndpoint, sttAPIKey, sttAPI, model)
		if err != nil {
			return err
		}
		segments, err = client.Transcribe(audioPath)
		if err != nil {
			return fmt.Errorf("stt via http: %w", err)
		}
	} else {
		log.Warn("No STT endpoint configured, generating placeholder transcript")
		segments = generatePlaceholderTranscript()
	}

	transcript := schema.RawTranscript{
		EpisodeID:    episodeID,
		AudioHash:    audioHash,
		Model:        model,
		ModelVersion: modelVersion,
		Segments:     segments,
		GeneratedAt:  time.Now(),
	}

	transcriptHash, err := hashing.HashStruct(transcript)
	if err != nil {
		return fmt.Errorf("hash transcript: %w", err)
	}
	transcript.TranscriptHash = transcriptHash

	if err := storage.WriteJSON(transcriptPath, transcript); err != nil {
		return fmt.Errorf("write transcript: %w", err)
	}

	log.Info("Wrote transcript: %s (%d segments)", transcriptPath, len(segments))
	return nil
}

func generatePlaceholderTranscript() []schema.TranscriptSegment {
	return []schema.TranscriptSegment{
		{
			Start:   0.0,
			End:     10.0,
			Text:    "This is a placeholder transcript segment.",
			Speaker: "Speaker 1",
		},
	}
}

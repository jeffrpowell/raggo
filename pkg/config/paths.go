package config

import "os"

func ResolveAudioDir(cfg *Config, corpusID, flagOverride string) string {
	if flagOverride != "" {
		return flagOverride
	}
	if cfg != nil {
		return cfg.Storage.Podcast.Audio
	}
	return "/var/lib/raggo/podcast/audio"
}

func ResolveTranscriptsDir(cfg *Config, corpusID, flagOverride string) string {
	if flagOverride != "" {
		return flagOverride
	}
	if cfg != nil {
		return cfg.Storage.Podcast.Transcripts
	}
	return "/var/lib/raggo/podcast/transcripts"
}

func ResolveChunksDir(cfg *Config, corpusID, flagOverride string) string {
	if flagOverride != "" {
		return flagOverride
	}
	if cfg != nil {
		return cfg.Storage.Podcast.Chunks
	}
	return "/var/lib/raggo/podcast/chunks"
}

func ResolveEmbeddingsDir(cfg *Config, corpusID, flagOverride string) string {
	if flagOverride != "" {
		return flagOverride
	}
	if cfg != nil {
		return cfg.Storage.Podcast.Embeddings
	}
	return "/var/lib/raggo/podcast/embeddings"
}

func ResolveIndexDir(cfg *Config, corpusID, flagOverride string) string {
	if flagOverride != "" {
		return flagOverride
	}
	if cfg != nil {
		return cfg.Storage.Podcast.Index
	}
	return "/var/lib/raggo/podcast/index"
}

func ResolveDocumentsSourcesDir(cfg *Config, corpusID, flagOverride string) string {
	if flagOverride != "" {
		return flagOverride
	}
	if cfg != nil {
		return cfg.Storage.Documents.Sources
	}
	return "/mnt/documents"
}

func ResolveDocumentsTextDir(cfg *Config, corpusID, flagOverride string) string {
	if flagOverride != "" {
		return flagOverride
	}
	if cfg != nil {
		return cfg.Storage.Documents.Text
	}
	return "/var/lib/raggo/documents/text"
}

func ResolveDocumentsChunksDir(cfg *Config, corpusID, flagOverride string) string {
	if flagOverride != "" {
		return flagOverride
	}
	if cfg != nil {
		return cfg.Storage.Documents.Chunks
	}
	return "/var/lib/raggo/documents/chunks"
}

func ResolveDocumentsEmbeddingsDir(cfg *Config, corpusID, flagOverride string) string {
	if flagOverride != "" {
		return flagOverride
	}
	if cfg != nil {
		return cfg.Storage.Documents.Embeddings
	}
	return "/var/lib/raggo/documents/embeddings"
}

func ResolveDocumentsIndexDir(cfg *Config, corpusID, flagOverride string) string {
	if flagOverride != "" {
		return flagOverride
	}
	if cfg != nil {
		return cfg.Storage.Documents.Index
	}
	return "/var/lib/raggo/documents/index"
}

func ResolveSTTEndpoint(cfg *Config, flagOverride string) string {
	if flagOverride != "" {
		return flagOverride
	}
	if cfg != nil {
		return cfg.Services.STTEndpoint
	}
	return "http://stt-service:8000/v1/audio/transcriptions"
}

func ResolveEmbedEndpoint(cfg *Config, flagOverride string) string {
	if flagOverride != "" {
		return flagOverride
	}
	if cfg != nil {
		return cfg.Services.EmbedEndpoint
	}
	return "http://embed-service:8001/v1/embeddings"
}

// ResolveSTTAPI returns the request format for stt_endpoint: "openai" (the
// default) or "raggo".
func ResolveSTTAPI(cfg *Config, flagOverride string) string {
	if flagOverride != "" {
		return flagOverride
	}
	if cfg != nil && cfg.Services.STTAPI != "" {
		return cfg.Services.STTAPI
	}
	return "openai"
}

func ResolveSTTModel(cfg *Config, flagOverride string) string {
	if flagOverride != "" {
		return flagOverride
	}
	if cfg != nil && cfg.Services.STTModel != "" {
		return cfg.Services.STTModel
	}
	return "whisper-1"
}

// ResolveEmbedAPI returns the request format for embed_endpoint: "openai"
// (the default) or "raggo".
func ResolveEmbedAPI(cfg *Config, flagOverride string) string {
	if flagOverride != "" {
		return flagOverride
	}
	if cfg != nil && cfg.Services.EmbedAPI != "" {
		return cfg.Services.EmbedAPI
	}
	return "openai"
}

func ResolveEmbedModel(cfg *Config, flagOverride string) string {
	if flagOverride != "" {
		return flagOverride
	}
	if cfg != nil && cfg.Services.EmbedModel != "" {
		return cfg.Services.EmbedModel
	}
	return "text-embedding-3-small"
}

func ResolveQdrantHost(cfg *Config, flagOverride string) string {
	if flagOverride != "" {
		return flagOverride
	}
	if cfg != nil {
		return cfg.Qdrant.Host
	}
	return "qdrant"
}

func ResolveQdrantPort(cfg *Config, flagOverride int) int {
	if flagOverride != 0 {
		return flagOverride
	}
	if cfg != nil {
		return cfg.Qdrant.Port
	}
	return 6334
}

func ResolvePodcastContextsDir(cfg *Config, corpusID, flagOverride string) string {
	if flagOverride != "" {
		return flagOverride
	}
	if cfg != nil && cfg.Storage.Podcast.Contexts != "" {
		return cfg.Storage.Podcast.Contexts
	}
	return "/var/lib/raggo/podcast/contexts"
}

func ResolveDocumentsContextsDir(cfg *Config, corpusID, flagOverride string) string {
	if flagOverride != "" {
		return flagOverride
	}
	if cfg != nil && cfg.Storage.Documents.Contexts != "" {
		return cfg.Storage.Documents.Contexts
	}
	return "/var/lib/raggo/documents/contexts"
}

func ResolveContextEndpoint(cfg *Config, flagOverride string) string {
	if flagOverride != "" {
		return flagOverride
	}
	if cfg != nil {
		return cfg.Services.ContextEndpoint
	}
	return ""
}

func ResolveContextModel(cfg *Config, flagOverride string) string {
	if flagOverride != "" {
		return flagOverride
	}
	if cfg != nil && cfg.Services.ContextModel != "" {
		return cfg.Services.ContextModel
	}
	return "qwen3.5-4b"
}

func ResolveContextBudgetChars(cfg *Config, flagOverride int) int {
	if flagOverride != 0 {
		return flagOverride
	}
	if cfg != nil && cfg.Services.ContextBudgetChars != 0 {
		return cfg.Services.ContextBudgetChars
	}
	return 12000
}

func ResolveTikaEndpoint(cfg *Config, flagOverride string) string {
	if flagOverride != "" {
		return flagOverride
	}
	if cfg != nil && cfg.Services.TikaEndpoint != "" {
		return cfg.Services.TikaEndpoint
	}
	return "http://localhost:9998"
}

// API keys have no flag override so they stay out of process listings and
// the orchestrator's logged stage args. A key set in config wins; otherwise
// the env var is used (e.g. from a compose .env file). Empty means no auth.
func resolveAPIKey(configValue, envVar string) string {
	if configValue != "" {
		return configValue
	}
	return os.Getenv(envVar)
}

func ResolveSTTAPIKey(cfg *Config) string {
	var v string
	if cfg != nil {
		v = cfg.Services.STTAPIKey
	}
	return resolveAPIKey(v, "RAGGO_STT_API_KEY")
}

func ResolveEmbedAPIKey(cfg *Config) string {
	var v string
	if cfg != nil {
		v = cfg.Services.EmbedAPIKey
	}
	return resolveAPIKey(v, "RAGGO_EMBED_API_KEY")
}

func ResolveContextAPIKey(cfg *Config) string {
	var v string
	if cfg != nil {
		v = cfg.Services.ContextAPIKey
	}
	return resolveAPIKey(v, "RAGGO_CONTEXT_API_KEY")
}

func ResolveTikaAPIKey(cfg *Config) string {
	var v string
	if cfg != nil {
		v = cfg.Services.TikaAPIKey
	}
	return resolveAPIKey(v, "RAGGO_TIKA_API_KEY")
}

func ResolveVisionAPIKey(cfg *Config) string {
	var v string
	if cfg != nil {
		v = cfg.Services.VisionAPIKey
	}
	return resolveAPIKey(v, "RAGGO_VISION_API_KEY")
}

func ResolveQdrantAPIKey(cfg *Config) string {
	var v string
	if cfg != nil {
		v = cfg.Qdrant.APIKey
	}
	return resolveAPIKey(v, "RAGGO_QDRANT_API_KEY")
}

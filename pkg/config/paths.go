package config

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
	return "http://stt-service:8000/transcribe"
}

func ResolveEmbedEndpoint(cfg *Config, flagOverride string) string {
	if flagOverride != "" {
		return flagOverride
	}
	if cfg != nil {
		return cfg.Services.EmbedEndpoint
	}
	return "http://embed-service:8001/embed"
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

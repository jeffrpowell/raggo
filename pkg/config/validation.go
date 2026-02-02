package config

import (
	"fmt"

	"github.com/robfig/cron/v3"
)

func Validate(cfg *Config) error {
	if cfg.Storage.Root == "" {
		return fmt.Errorf("storage.root is required")
	}

	if cfg.Qdrant.Host == "" {
		return fmt.Errorf("qdrant.host is required")
	}

	if cfg.Qdrant.Port == 0 {
		return fmt.Errorf("qdrant.port is required")
	}

	if cfg.Qdrant.MetadataCollection == "" {
		return fmt.Errorf("qdrant.metadata_collection is required")
	}

	if cfg.Services.STTEndpoint == "" {
		return fmt.Errorf("services.stt_endpoint is required")
	}

	if cfg.Services.EmbedEndpoint == "" {
		return fmt.Errorf("services.embed_endpoint is required")
	}

	if cfg.Orchestrator.RetryLimit < 1 {
		return fmt.Errorf("orchestrator.retry_limit must be at least 1")
	}

	if cfg.Orchestrator.RetryDelaySeconds < 0 {
		return fmt.Errorf("orchestrator.retry_delay_seconds cannot be negative")
	}

	if cfg.Orchestrator.LogDir == "" {
		return fmt.Errorf("orchestrator.log_dir is required")
	}

	corpusIDs := make(map[string]bool)
	for _, corpus := range cfg.Corpora {
		if corpus.ID == "" {
			return fmt.Errorf("corpus ID is required")
		}

		if corpusIDs[corpus.ID] {
			return fmt.Errorf("duplicate corpus ID: %s", corpus.ID)
		}
		corpusIDs[corpus.ID] = true

		if corpus.Type == "" {
			return fmt.Errorf("corpus %s: type is required", corpus.ID)
		}

		if corpus.Type != "podcast" && corpus.Type != "documents" {
			return fmt.Errorf("corpus %s: type must be 'podcast' or 'documents', got: %s", corpus.ID, corpus.Type)
		}

		if corpus.Mode == "" {
			return fmt.Errorf("corpus %s: mode is required", corpus.ID)
		}

		if corpus.Mode != "one-shot" && corpus.Mode != "scheduled" && corpus.Mode != "watch" {
			return fmt.Errorf("corpus %s: mode must be 'one-shot', 'scheduled', or 'watch', got: %s", corpus.ID, corpus.Mode)
		}

		if corpus.Mode == "scheduled" {
			if corpus.Schedule == "" {
				return fmt.Errorf("corpus %s: schedule is required for scheduled mode", corpus.ID)
			}
			parser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
			if _, err := parser.Parse(corpus.Schedule); err != nil {
				return fmt.Errorf("corpus %s: invalid schedule format: %w", corpus.ID, err)
			}
		}

		if corpus.Mode == "watch" {
			if corpus.WatchIntervalSeconds <= 0 {
				return fmt.Errorf("corpus %s: watch_interval_seconds must be greater than 0", corpus.ID)
			}
		}

		if err := validateCorpusConfig(corpus); err != nil {
			return fmt.Errorf("corpus %s: %w", corpus.ID, err)
		}
	}

	return nil
}

func validateCorpusConfig(corpus CorpusConfig) error {
	if corpus.Type == "podcast" {
		if _, ok := corpus.Config["feed_url"]; !ok {
			return fmt.Errorf("feed_url is required for podcast corpus")
		}
		if _, ok := corpus.Config["collection"]; !ok {
			return fmt.Errorf("collection is required for podcast corpus")
		}
		if _, ok := corpus.Config["corpus"]; !ok {
			return fmt.Errorf("corpus is required for podcast corpus")
		}
	}

	if corpus.Type == "documents" {
		if _, ok := corpus.Config["root_path"]; !ok {
			return fmt.Errorf("root_path is required for documents corpus")
		}
		if _, ok := corpus.Config["collection"]; !ok {
			return fmt.Errorf("collection is required for documents corpus")
		}
		if _, ok := corpus.Config["corpus"]; !ok {
			return fmt.Errorf("corpus is required for documents corpus")
		}
	}

	return nil
}

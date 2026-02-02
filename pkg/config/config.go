package config

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config file: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	if err := expandPaths(&cfg); err != nil {
		return nil, fmt.Errorf("expand paths: %w", err)
	}

	if err := Validate(&cfg); err != nil {
		return nil, fmt.Errorf("validate config: %w", err)
	}

	return &cfg, nil
}

func expandPaths(cfg *Config) error {
	root := cfg.Storage.Root

	cfg.Storage.Podcast.Audio = expandVar(cfg.Storage.Podcast.Audio, "root", root)
	cfg.Storage.Podcast.Transcripts = expandVar(cfg.Storage.Podcast.Transcripts, "root", root)
	cfg.Storage.Podcast.Chunks = expandVar(cfg.Storage.Podcast.Chunks, "root", root)
	cfg.Storage.Podcast.Embeddings = expandVar(cfg.Storage.Podcast.Embeddings, "root", root)
	cfg.Storage.Podcast.Index = expandVar(cfg.Storage.Podcast.Index, "root", root)

	cfg.Storage.Documents.Sources = expandVar(cfg.Storage.Documents.Sources, "root", root)
	cfg.Storage.Documents.Text = expandVar(cfg.Storage.Documents.Text, "root", root)
	cfg.Storage.Documents.Chunks = expandVar(cfg.Storage.Documents.Chunks, "root", root)
	cfg.Storage.Documents.Embeddings = expandVar(cfg.Storage.Documents.Embeddings, "root", root)
	cfg.Storage.Documents.Index = expandVar(cfg.Storage.Documents.Index, "root", root)

	return nil
}

func expandVar(path, varName, value string) string {
	return strings.ReplaceAll(path, "${"+varName+"}", value)
}

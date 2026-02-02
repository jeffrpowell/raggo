package main

import (
	"github.com/jeffrpowell/raggo/pkg/config"
	"github.com/jeffrpowell/raggo/pkg/logging"
	"github.com/jeffrpowell/raggo/pkg/pipeline"
)

func runOneShotMode(executor *pipeline.Executor, corpora []config.CorpusConfig, logger *logging.Logger) {
	logger.Info("Starting one-shot mode with %d corpora", len(corpora))

	for _, corpus := range corpora {
		if err := executor.ExecutePipelineWithRetry(corpus); err != nil {
			logger.Error("Corpus failed: %s, error: %v", corpus.ID, err)
		}
	}

	logger.Info("One-shot mode complete")
}

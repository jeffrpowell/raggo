package main

import (
	"github.com/jeffrpowell/raggo/pkg/config"
	"github.com/jeffrpowell/raggo/pkg/logging"
	"github.com/jeffrpowell/raggo/pkg/pipeline"
	"github.com/robfig/cron/v3"
)

func runScheduledMode(executor *pipeline.Executor, corpora []config.CorpusConfig, logger *logging.Logger) {
	logger.Info("Starting scheduled mode with %d corpora", len(corpora))

	c := cron.New()

	for _, corpus := range corpora {
		corpus := corpus
		_, err := c.AddFunc(corpus.Schedule, func() {
			logger.Info("Scheduled trigger for corpus: %s", corpus.ID)
			if err := executor.ExecutePipelineWithRetry(corpus); err != nil {
				logger.Error("Scheduled run failed for corpus: %s, error: %v", corpus.ID, err)
			}
		})
		if err != nil {
			logger.Error("Failed to schedule corpus: %s, error: %v", corpus.ID, err)
		}
	}

	c.Start()

	select {}
}

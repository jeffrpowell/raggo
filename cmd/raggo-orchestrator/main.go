package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/jeffrpowell/raggo/pkg/config"
	"github.com/jeffrpowell/raggo/pkg/logging"
	"github.com/jeffrpowell/raggo/pkg/pipeline"
	"github.com/jeffrpowell/raggo/pkg/state"
)

func main() {
	var configPath string
	flag.StringVar(&configPath, "config", "/etc/raggo/raggo.yml", "Path to config")
	flag.Parse()

	cfg, err := config.Load(configPath)
	if err != nil {
		log.Fatalf("Load config: %v", err)
	}

	logger := logging.New("raggo-orchestrator")

	stateManager, err := state.NewQdrantStateManager(cfg.Qdrant)
	if err != nil {
		log.Fatalf("Init state manager: %v", err)
	}
	defer stateManager.Close()

	executor := pipeline.NewExecutor(cfg, stateManager, logger)

	var wg sync.WaitGroup

	oneshotCorpora := filterByMode(cfg.Corpora, "one-shot")
	if len(oneshotCorpora) > 0 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			runOneShotMode(executor, oneshotCorpora, logger)
		}()
	}

	scheduledCorpora := filterByMode(cfg.Corpora, "scheduled")
	if len(scheduledCorpora) > 0 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			runScheduledMode(executor, scheduledCorpora, logger)
		}()
	}

	watchCorpora := filterByMode(cfg.Corpora, "watch")
	if len(watchCorpora) > 0 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			runWatchMode(executor, watchCorpora, logger, stateManager)
		}()
	}

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		logger.Info("Received shutdown signal")
		os.Exit(0)
	}()

	wg.Wait()
}

func filterByMode(corpora []config.CorpusConfig, mode string) []config.CorpusConfig {
	var result []config.CorpusConfig
	for _, corpus := range corpora {
		if corpus.Mode == mode {
			result = append(result, corpus)
		}
	}
	return result
}

package pipeline

import (
	"bytes"
	"fmt"
	"os/exec"
	"time"

	"github.com/jeffrpowell/raggo/pkg/config"
	"github.com/jeffrpowell/raggo/pkg/logging"
	"github.com/jeffrpowell/raggo/pkg/state"
)

type Executor struct {
	cfg          *config.Config
	stateManager state.StateManager
	logger       *logging.Logger
}

func NewExecutor(cfg *config.Config, stateManager state.StateManager, logger *logging.Logger) *Executor {
	return &Executor{
		cfg:          cfg,
		stateManager: stateManager,
		logger:       logger,
	}
}

func (e *Executor) ExecutePipeline(corpus config.CorpusConfig) error {
	runID, err := e.stateManager.StartRun(corpus.ID)
	if err != nil {
		return fmt.Errorf("start run: %w", err)
	}

	ctx := &ExecutionContext{
		Config:       e.cfg,
		Corpus:       corpus,
		StateManager: e.stateManager,
		Logger:       e.logger,
		RunID:        runID,
	}

	var pipeline *Pipeline
	switch corpus.Type {
	case "podcast":
		pipeline = BuildPodcastPipeline(e.cfg, corpus)
	case "documents":
		pipeline = BuildDocumentPipeline(e.cfg, corpus)
	default:
		return fmt.Errorf("unknown corpus type: %s", corpus.Type)
	}

	for _, stage := range pipeline.Stages {
		skip, err := stage.CheckSkip(ctx)
		if err != nil {
			return fmt.Errorf("check skip %s: %w", stage.Name, err)
		}
		if skip {
			e.logger.Info("Skipping stage (outputs exist): %s", stage.Name)
			continue
		}

		e.stateManager.UpdateRunProgress(runID, stage.Name, 0)

		e.logger.Info("Executing stage: %s", stage.Name)
		if err := e.executeStage(ctx, stage); err != nil {
			e.stateManager.CompleteRun(runID, "failed", err.Error())
			return fmt.Errorf("stage %s: %w", stage.Name, err)
		}
	}

	if !e.cfg.Storage.RetainIntermediates {
		e.logger.Info("Cleanup intermediates: retain_intermediates=false")
	}

	e.stateManager.CompleteRun(runID, "success", "")
	return nil
}

func (e *Executor) executeStage(ctx *ExecutionContext, stage Stage) error {
	args := stage.BuildArgs(ctx)
	cmd := exec.Command(stage.Binary, args...)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start: %w", err)
	}

	if err := cmd.Wait(); err != nil {
		e.logger.Error("Stage failed: %s", stage.Name)
		e.logger.Error("stdout: %s", stdout.String())
		e.logger.Error("stderr: %s", stderr.String())
		return fmt.Errorf("execution failed: %w", err)
	}

	e.logger.Info("Stage completed: %s", stage.Name)
	return nil
}

func (e *Executor) ExecutePipelineWithRetry(corpus config.CorpusConfig) error {
	retryLimit := e.cfg.Orchestrator.RetryLimit
	retryDelay := time.Duration(e.cfg.Orchestrator.RetryDelaySeconds) * time.Second

	var lastErr error
	for attempt := 1; attempt <= retryLimit; attempt++ {
		e.logger.Info("Pipeline attempt %d/%d for corpus: %s", attempt, retryLimit, corpus.ID)

		err := e.ExecutePipeline(corpus)
		if err == nil {
			e.logger.Info("Pipeline succeeded for corpus: %s (attempt %d)", corpus.ID, attempt)
			return nil
		}

		lastErr = err
		e.logger.Warn("Pipeline failed for corpus: %s (attempt %d): %v", corpus.ID, attempt, err)

		if attempt < retryLimit {
			e.logger.Info("Retrying after %v delay...", retryDelay)
			time.Sleep(retryDelay)
		}
	}

	return fmt.Errorf("pipeline failed after %d attempts: %w", retryLimit, lastErr)
}

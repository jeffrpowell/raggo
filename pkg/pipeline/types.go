package pipeline

import (
	"github.com/jeffrpowell/raggo/pkg/config"
	"github.com/jeffrpowell/raggo/pkg/logging"
	"github.com/jeffrpowell/raggo/pkg/state"
)

type Stage struct {
	Name      string
	Binary    string
	BuildArgs func(ctx *ExecutionContext) []string
	CheckSkip func(ctx *ExecutionContext) (bool, error)
}

type Pipeline struct {
	Stages []Stage
}

type ExecutionContext struct {
	Config       *config.Config
	Corpus       config.CorpusConfig
	StateManager state.StateManager
	Logger       *logging.Logger
	RunID        string
	TempDir      string
}

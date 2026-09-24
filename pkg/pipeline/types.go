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
	// Items, when set, runs the binary once per returned item with
	// ItemFlag <id> and the item's Args appended (e.g. -document-id, -episode-id).
	Items    func(ctx *ExecutionContext) ([]Item, error)
	ItemFlag string
}

// Item is one unit of per-ID stage work.
type Item struct {
	ID   string
	Args []string
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

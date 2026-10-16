package pipeline

import "github.com/jeffrpowell/raggo/pkg/config"

// describeStage stores the corpus `description` in its Qdrant collection's
// metadata so query-side tools can pick a collection. It runs once per
// pipeline run (no items), so editing the description takes effect without
// re-indexing. Skipped when the corpus has no description.
func describeStage(corpus config.CorpusConfig) Stage {
	return Stage{
		Name:   "describe",
		Binary: "/usr/local/bin/raggo-describe-collection",
		BuildArgs: func(ctx *ExecutionContext) []string {
			return []string{
				"-config", "/etc/raggo/raggo.yml",
				"-collection", corpus.Config["collection"].(string),
				"-description", corpusDescription(corpus),
			}
		},
		CheckSkip: func(ctx *ExecutionContext) (bool, error) {
			return corpusDescription(corpus) == "", nil
		},
	}
}

func corpusDescription(corpus config.CorpusConfig) string {
	description, _ := corpus.Config["description"].(string)
	return description
}

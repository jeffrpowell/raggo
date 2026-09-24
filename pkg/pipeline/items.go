package pipeline

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/jeffrpowell/raggo/pkg/schema"
	"github.com/jeffrpowell/raggo/pkg/storage"
)

// Item sources are evaluated when a stage starts, so they see the artifacts
// the previous stage just produced. A missing directory yields no items, which
// makes the stage a no-op.

// idsFromFiles lists IDs from <dir>/<id><suffix> files, skipping names that
// end in any of the excluded suffixes (e.g. ".normalized.json").
func idsFromFiles(dir, suffix string, exclude ...string) func(ctx *ExecutionContext) ([]Item, error) {
	return func(ctx *ExecutionContext) ([]Item, error) {
		files, err := filepath.Glob(filepath.Join(dir, "*"+suffix))
		if err != nil {
			return nil, err
		}
		var items []Item
	next:
		for _, f := range files {
			name := filepath.Base(f)
			for _, ex := range exclude {
				if strings.HasSuffix(name, ex) {
					continue next
				}
			}
			items = append(items, Item{ID: strings.TrimSuffix(name, suffix)})
		}
		return sortItems(items), nil
	}
}

// documentManifestItems lists every scanned document with its source path,
// read from the scan stage's <dir>/*.jsonl manifests.
func documentManifestItems(dir string) func(ctx *ExecutionContext) ([]Item, error) {
	return func(ctx *ExecutionContext) ([]Item, error) {
		files, err := filepath.Glob(filepath.Join(dir, "*.jsonl"))
		if err != nil {
			return nil, err
		}
		seen := map[string]bool{}
		var items []Item
		for _, f := range files {
			err := storage.ReadJSONL(f, func(line []byte) error {
				var m schema.DocumentManifest
				if err := json.Unmarshal(line, &m); err != nil {
					return err
				}
				if m.DocumentID == "" || m.AbsolutePath == "" || seen[m.DocumentID] {
					return nil
				}
				seen[m.DocumentID] = true
				items = append(items, Item{ID: m.DocumentID, Args: []string{"-file-path", m.AbsolutePath}})
				return nil
			})
			if err != nil {
				return nil, err
			}
		}
		return sortItems(items), nil
	}
}

// audioItems lists downloaded episodes with their audio file, read from the
// download stage's <dir>/<episode_id>.json metadata. Other JSON files in the
// directory (feed metadata) lack these fields and are skipped.
func audioItems(dir string) func(ctx *ExecutionContext) ([]Item, error) {
	return func(ctx *ExecutionContext) ([]Item, error) {
		files, err := filepath.Glob(filepath.Join(dir, "*.json"))
		if err != nil {
			return nil, err
		}
		var items []Item
		for _, f := range files {
			data, err := os.ReadFile(f)
			if err != nil {
				return nil, err
			}
			var m schema.AudioMetadata
			if json.Unmarshal(data, &m) != nil || m.EpisodeID == "" || m.FilePath == "" {
				continue
			}
			items = append(items, Item{ID: m.EpisodeID, Args: []string{"-audio-path", m.FilePath}})
		}
		return sortItems(items), nil
	}
}

func sortItems(items []Item) []Item {
	sort.Slice(items, func(i, j int) bool { return items[i].ID < items[j].ID })
	return items
}

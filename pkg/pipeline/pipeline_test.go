package pipeline

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/jeffrpowell/raggo/pkg/config"
)

func stageNames(p *Pipeline) []string {
	var names []string
	for _, s := range p.Stages {
		names = append(names, s.Name)
	}
	return names
}

func TestContextualizeStageOrder(t *testing.T) {
	cfg := &config.Config{}
	corpus := config.CorpusConfig{ID: "c", Config: map[string]interface{}{"feed_url": "https://example.com/feed.xml"}}

	for name, p := range map[string]*Pipeline{
		"documents": BuildDocumentPipeline(cfg, corpus),
		"podcast":   BuildPodcastPipeline(cfg, corpus),
	} {
		names := stageNames(p)
		c, x, e := slices.Index(names, "chunk"), slices.Index(names, "contextualize"), slices.Index(names, "embed")
		if !(c >= 0 && c+1 == x && x+1 == e) {
			t.Errorf("%s: contextualize must sit between chunk and embed, got %v", name, names)
		}
	}
}

func TestContextualizeOptOut(t *testing.T) {
	cfg := &config.Config{}
	for _, tc := range []struct {
		cfg  map[string]interface{}
		skip bool
	}{
		{map[string]interface{}{}, false},
		{map[string]interface{}{"contextualize": true}, false},
		{map[string]interface{}{"contextualize": false}, true},
	} {
		p := BuildDocumentPipeline(cfg, config.CorpusConfig{ID: "c", Config: tc.cfg})
		for _, s := range p.Stages {
			if s.Name != "contextualize" {
				continue
			}
			skip, err := s.CheckSkip(nil)
			if err != nil || skip != tc.skip {
				t.Errorf("config %v: skip=%v err=%v, want skip=%v", tc.cfg, skip, err, tc.skip)
			}
		}
	}
}

func TestContextBudgetOverridePassed(t *testing.T) {
	p := BuildDocumentPipeline(&config.Config{}, config.CorpusConfig{ID: "c", Config: map[string]interface{}{"context_budget_chars": 40000}})
	for _, s := range p.Stages {
		if s.Name == "contextualize" {
			args := s.BuildArgs(nil)
			i := slices.Index(args, "-context-budget-chars")
			if i < 0 || args[i+1] != "40000" {
				t.Errorf("budget override not passed: %v", args)
			}
		}
	}
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

func TestEveryIDStageFansOutOverUpstreamArtifacts(t *testing.T) {
	root := t.TempDir()
	cfg := &config.Config{}
	cfg.Storage.Documents.Text = filepath.Join(root, "docs/text")
	cfg.Storage.Documents.Chunks = filepath.Join(root, "docs/chunks")
	cfg.Storage.Podcast.Audio = filepath.Join(root, "pod/audio")
	cfg.Storage.Podcast.Transcripts = filepath.Join(root, "pod/transcripts")
	cfg.Storage.Podcast.Chunks = filepath.Join(root, "pod/chunks")

	// Documents: scan manifest (with a duplicate), extracted text, chunks.
	writeFile(t, filepath.Join(cfg.Storage.Documents.Text, "roothash.jsonl"),
		`{"document_id":"d2","absolute_path":"/mnt/documents/b.pdf"}`+"\n"+
			`{"document_id":"d1","absolute_path":"/mnt/documents/a.md"}`+"\n"+
			`{"document_id":"d1","absolute_path":"/mnt/documents/a.md"}`+"\n")
	writeFile(t, filepath.Join(cfg.Storage.Documents.Text, "d1.json"), "{}")
	writeFile(t, filepath.Join(cfg.Storage.Documents.Chunks, "d1.jsonl"), "")
	writeFile(t, filepath.Join(cfg.Storage.Documents.Chunks, "notes.txt"), "")

	// Podcast: feed metadata and audio metadata share the audio dir.
	writeFile(t, filepath.Join(cfg.Storage.Podcast.Audio, "feedhash.json"), `{"title":"Show","feed_url":"u"}`)
	writeFile(t, filepath.Join(cfg.Storage.Podcast.Audio, "feedhash.jsonl"), `{"episode_id":"e1"}`)
	writeFile(t, filepath.Join(cfg.Storage.Podcast.Audio, "e1.json"), `{"episode_id":"e1","file_path":"/a/e1.mp3"}`)
	writeFile(t, filepath.Join(cfg.Storage.Podcast.Audio, "e2.json"), `{"episode_id":"e2","file_path":"/a/e2.mp3"}`)
	writeFile(t, filepath.Join(cfg.Storage.Podcast.Transcripts, "e1.json"), "{}")
	writeFile(t, filepath.Join(cfg.Storage.Podcast.Transcripts, "e2.json"), "{}")
	writeFile(t, filepath.Join(cfg.Storage.Podcast.Transcripts, "e1.normalized.json"), "{}")
	writeFile(t, filepath.Join(cfg.Storage.Podcast.Chunks, "e1.jsonl"), "")

	corpus := config.CorpusConfig{ID: "c", Config: map[string]interface{}{
		"feed_url": "https://example.com/feed.xml", "root_path": "/mnt/documents", "collection": "col", "corpus": "c1",
	}}

	want := map[string]map[string][]Item{
		"documents": {
			"extract": {{ID: "d1", Args: []string{"-file-path", "/mnt/documents/a.md"}}, {ID: "d2", Args: []string{"-file-path", "/mnt/documents/b.pdf"}}},
			"chunk":   {{ID: "d1"}},
			"embed":   {{ID: "d1"}},
			"index":   {{ID: "d1"}},
		},
		"podcast": {
			"stt":       {{ID: "e1", Args: []string{"-audio-path", "/a/e1.mp3"}}, {ID: "e2", Args: []string{"-audio-path", "/a/e2.mp3"}}},
			"normalize": {{ID: "e1"}, {ID: "e2"}},
			"chunk":     {{ID: "e1"}},
			"embed":     {{ID: "e1"}},
			"index":     {{ID: "e1"}},
		},
	}
	flags := map[string]string{"documents": "-document-id", "podcast": "-episode-id"}
	pipelines := map[string]*Pipeline{
		"documents": BuildDocumentPipeline(cfg, corpus),
		"podcast":   BuildPodcastPipeline(cfg, corpus),
	}

	for kind, p := range pipelines {
		for _, s := range p.Stages {
			expected, needsIDs := want[kind][s.Name]
			if !needsIDs {
				if s.Items != nil {
					t.Errorf("%s/%s: unexpected fan-out", kind, s.Name)
				}
				continue
			}
			if s.Items == nil || s.ItemFlag != flags[kind] {
				t.Errorf("%s/%s: expected fan-out with %s", kind, s.Name, flags[kind])
				continue
			}
			got, err := s.Items(nil)
			if err != nil {
				t.Errorf("%s/%s: %v", kind, s.Name, err)
				continue
			}
			if !slices.EqualFunc(got, expected, func(a, b Item) bool { return a.ID == b.ID && slices.Equal(a.Args, b.Args) }) {
				t.Errorf("%s/%s: got %+v, want %+v", kind, s.Name, got, expected)
			}
		}
	}
}

func TestItemSourcesTolerateMissingDirs(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "nope")
	for name, src := range map[string]func(*ExecutionContext) ([]Item, error){
		"files":     idsFromFiles(missing, ".jsonl"),
		"manifests": documentManifestItems(missing),
		"audio":     audioItems(missing),
	} {
		items, err := src(nil)
		if err != nil || len(items) != 0 {
			t.Errorf("%s: items=%v err=%v", name, items, err)
		}
	}
}

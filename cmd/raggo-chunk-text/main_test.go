package main

import (
	"strings"
	"testing"
	"time"
)

func TestCreateTextChunksTerminates(t *testing.T) {
	cases := map[string]string{
		"shorter than chunk": strings.Repeat("Hello world. ", 20),
		"several chunks":     strings.Repeat("Hello world. ", 300),
		"no sentence enders": strings.Repeat("x", 3500),
	}

	for name, text := range cases {
		t.Run(name, func(t *testing.T) {
			done := make(chan []string, 1)
			go func() {
				var texts []string
				for _, c := range createTextChunks("doc", text, 1000, 100) {
					texts = append(texts, c.Text)
				}
				done <- texts
			}()

			var texts []string
			select {
			case texts = <-done:
			case <-time.After(5 * time.Second):
				t.Fatal("createTextChunks did not terminate")
			}

			normalized := normalizeWhitespace(text)
			if len(texts) == 0 {
				t.Fatal("expected at least one chunk")
			}
			if !strings.HasSuffix(normalized, texts[len(texts)-1]) {
				t.Error("last chunk does not end at end of text")
			}
			// Sentence-boundary chunks are at least ~800 chars, so each step
			// advances at least ~700 past the 100-char overlap.
			if limit := len(normalized)/700 + 1; len(texts) > limit {
				t.Errorf("got %d chunks for %d chars, want at most %d", len(texts), len(normalized), limit)
			}
		})
	}
}

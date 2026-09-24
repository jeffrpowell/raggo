package contextual

import (
	"context"
	"fmt"
	"strings"
	"testing"
)

type fakeCompleter struct {
	prompts []string
}

func (f *fakeCompleter) Complete(_ context.Context, _, user string) (string, error) {
	f.prompts = append(f.prompts, user)
	return fmt.Sprintf("<think>ignored</think> ctx-%d ", len(f.prompts)), nil
}

func makePieces(n, size int) []Piece {
	pieces := make([]Piece, n)
	for i := range pieces {
		text := strings.Repeat(fmt.Sprintf("p%d ", i), size/4)
		pieces[i] = Piece{ID: fmt.Sprintf("c%d", i), Text: text, TextHash: fmt.Sprintf("h%d", i)}
	}
	return pieces
}

func TestModeSelection(t *testing.T) {
	pieces := makePieces(3, 100)
	if got := Mode(pieces, 10000); got != ModeFull {
		t.Errorf("small doc: got %s, want %s", got, ModeFull)
	}
	if got := Mode(pieces, 150); got != ModeWindowed {
		t.Errorf("large doc: got %s, want %s", got, ModeWindowed)
	}
}

func TestFullModeSharesPrefixAndPutsChunkLast(t *testing.T) {
	pieces := makePieces(4, 100)
	fc := &fakeCompleter{}

	results, err := Contextualize(context.Background(), fc, Meta{Title: "Doc"}, pieces, "", 10000, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != len(pieces) {
		t.Fatalf("got %d results, want %d", len(results), len(pieces))
	}
	for i, r := range results {
		if r.ChunkID != pieces[i].ID || r.TextHash != pieces[i].TextHash || r.Mode != ModeFull {
			t.Errorf("result %d mismatch: %+v", i, r)
		}
		if strings.Contains(r.Context, "think") || r.Context != strings.TrimSpace(r.Context) {
			t.Errorf("context not cleaned: %q", r.Context)
		}
	}

	prefix := commonPrefix(fc.prompts)
	if !strings.Contains(prefix, "</document>") {
		t.Errorf("shared prefix should contain the whole document, got %q", prefix)
	}
	for i, p := range fc.prompts {
		if strings.Index(p, "<chunk>") < strings.Index(p, "</document>") {
			t.Errorf("prompt %d: chunk must come after document", i)
		}
		if !strings.Contains(p, pieces[i].Text) {
			t.Errorf("prompt %d missing its chunk", i)
		}
	}
}

func TestWindowedModeGroupsShareExcerpt(t *testing.T) {
	pieces := makePieces(12, 100)
	fc := &fakeCompleter{}

	results, err := Contextualize(context.Background(), fc, Meta{}, pieces, "SYNOPSIS", 500, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != len(pieces) {
		t.Fatalf("got %d results, want %d", len(results), len(pieces))
	}

	excerpts := map[string]int{}
	for i, p := range fc.prompts {
		if !strings.Contains(p, "SYNOPSIS") {
			t.Errorf("prompt %d missing synopsis", i)
		}
		ex := between(p, "<document_excerpt>", "</document_excerpt>")
		if !strings.Contains(ex, strings.TrimSpace(pieces[i].Text)) {
			t.Errorf("prompt %d: excerpt does not contain its chunk", i)
		}
		if len(ex) > 500 {
			t.Errorf("prompt %d: excerpt exceeds budget: %d chars", i, len(ex))
		}
		excerpts[ex]++
	}
	if len(excerpts) < 2 || len(excerpts) >= len(pieces) {
		t.Errorf("expected chunks grouped into a few windows, got %d distinct excerpts", len(excerpts))
	}
	for i := 1; i < len(fc.prompts); i++ {
		a := between(fc.prompts[i-1], "<document_excerpt>", "</document_excerpt>")
		b := between(fc.prompts[i], "<document_excerpt>", "</document_excerpt>")
		if a != b {
			continue
		}
		pa := fc.prompts[i-1][:strings.Index(fc.prompts[i-1], "<chunk>")]
		pb := fc.prompts[i][:strings.Index(fc.prompts[i], "<chunk>")]
		if pa != pb {
			t.Errorf("prompts %d and %d share an excerpt but not a byte-identical prefix", i-1, i)
		}
	}
}

func TestBuildSynopsisRecursesUntilFits(t *testing.T) {
	pieces := makePieces(20, 100)
	fc := &fakeCompleter{}

	syn, err := BuildSynopsis(context.Background(), fc, Meta{Title: "Big"}, pieces, 300)
	if err != nil {
		t.Fatal(err)
	}
	if syn == "" {
		t.Fatal("empty synopsis")
	}
	last := fc.prompts[len(fc.prompts)-1]
	if !strings.Contains(last, reduceInstruction) {
		t.Errorf("final call should be the reduce prompt")
	}
	if len(fc.prompts) <= len(groupByBudget(pieces, 300)) {
		t.Errorf("expected map calls plus at least one reduce, got %d calls", len(fc.prompts))
	}
}

func TestJoinPiecesDropsOverlap(t *testing.T) {
	pieces := []Piece{
		{Text: "The quick brown fox jumps over the lazy sleeping dog."},
		{Text: "jumps over the lazy sleeping dog. Then it ran away quickly."},
		{Text: "Unrelated start."},
	}
	got := joinPieces(pieces)
	want := "The quick brown fox jumps over the lazy sleeping dog. Then it ran away quickly. Unrelated start."
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func commonPrefix(ss []string) string {
	if len(ss) == 0 {
		return ""
	}
	p := ss[0]
	for _, s := range ss[1:] {
		for !strings.HasPrefix(s, p) {
			p = p[:len(p)-1]
		}
	}
	return p
}

func between(s, open, close string) string {
	i := strings.Index(s, open)
	j := strings.Index(s, close)
	if i < 0 || j < i {
		return ""
	}
	return s[i+len(open) : j]
}

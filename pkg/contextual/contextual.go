// Package contextual implements Anthropic-style contextual retrieval
// (https://www.anthropic.com/engineering/contextual-retrieval) for models
// whose context window cannot hold a whole document.
//
// Documents that fit in the character budget use the article's method: the
// full document is the prompt prefix for every chunk. Larger documents use a
// cached synopsis plus a local excerpt around each group of chunks. In both
// modes, chunks are processed in order and the chunk is placed last in the
// prompt, so consecutive calls share an identical prefix and llama-server can
// reuse its KV cache.
package contextual

import (
	"context"
	"fmt"
	"regexp"
	"strings"
)

const (
	PromptVersion = "v1"

	ModeFull     = "full"
	ModeWindowed = "windowed"

	maxDescriptionChars = 1000
	maxOverlapScan      = 500
	minOverlap          = 20
	maxReduceDepth      = 6
)

const systemPrompt = "You write brief context passages that situate a chunk within its source document to improve search retrieval."

const chunkInstruction = "Please give a short succinct context to situate this chunk within the overall document for the purposes of improving search retrieval of the chunk. Answer only with the succinct context and nothing else."

const windowedChunkInstruction = "Please give a short succinct context to situate this chunk within the overall document (use the summary and the surrounding excerpt) for the purposes of improving search retrieval of the chunk. Answer only with the succinct context and nothing else."

const mapInstruction = "Write concise notes (at most 100 words) on this section: the topics, named entities, and what the section establishes. Answer only with the notes."

const reduceInstruction = "These are notes on consecutive sections of one document. Write a synopsis of the whole document (at most 300 words) covering its purpose, structure, and key entities. Answer only with the synopsis."

var thinkBlock = regexp.MustCompile(`(?s)<think>.*?</think>`)

// Completer is satisfied by *llm.Client.
type Completer interface {
	Complete(ctx context.Context, system, user string) (string, error)
}

type Meta struct {
	Title       string
	Source      string
	Description string
}

type Piece struct {
	ID       string
	Text     string
	TextHash string
}

type Result struct {
	ChunkID  string
	TextHash string
	Context  string
	Mode     string
}

// Mode reports which strategy Contextualize will use for these pieces.
func Mode(pieces []Piece, budgetChars int) string {
	if len(joinPieces(pieces)) <= budgetChars {
		return ModeFull
	}
	return ModeWindowed
}

// BuildSynopsis summarizes a document too large for the budget via
// map-reduce over budget-sized windows.
func BuildSynopsis(ctx context.Context, c Completer, meta Meta, pieces []Piece, budgetChars int) (string, error) {
	var notes []string
	for i, group := range groupByBudget(pieces, budgetChars) {
		user := metaBlock(meta) +
			"<document_section>\n" + joinPieces(group) + "\n</document_section>\n\n" +
			mapInstruction
		out, err := complete(ctx, c, user)
		if err != nil {
			return "", fmt.Errorf("map window %d: %w", i, err)
		}
		notes = append(notes, out)
	}
	return reduceNotes(ctx, c, meta, notes, budgetChars, 0)
}

func reduceNotes(ctx context.Context, c Completer, meta Meta, notes []string, budgetChars, depth int) (string, error) {
	joined := strings.Join(notes, "\n\n")
	if len(joined) <= budgetChars || depth >= maxReduceDepth || len(notes) == 1 {
		user := metaBlock(meta) +
			"<section_notes>\n" + truncate(joined, budgetChars) + "\n</section_notes>\n\n" +
			reduceInstruction
		return complete(ctx, c, user)
	}

	var merged []string
	var batch []string
	size := 0
	flush := func() error {
		if len(batch) == 0 {
			return nil
		}
		user := metaBlock(meta) +
			"<section_notes>\n" + strings.Join(batch, "\n\n") + "\n</section_notes>\n\n" +
			mapInstruction
		out, err := complete(ctx, c, user)
		if err != nil {
			return err
		}
		merged = append(merged, out)
		batch, size = nil, 0
		return nil
	}
	for _, n := range notes {
		if size+len(n) > budgetChars && len(batch) > 0 {
			if err := flush(); err != nil {
				return "", fmt.Errorf("reduce depth %d: %w", depth, err)
			}
		}
		batch = append(batch, n)
		size += len(n) + 2
	}
	if err := flush(); err != nil {
		return "", fmt.Errorf("reduce depth %d: %w", depth, err)
	}
	return reduceNotes(ctx, c, meta, merged, budgetChars, depth+1)
}

// Contextualize generates a context passage for every piece, in order.
// synopsis is only used in windowed mode. progress, if non-nil, is called
// after each piece.
func Contextualize(ctx context.Context, c Completer, meta Meta, pieces []Piece, synopsis string, budgetChars int, progress func(done, total int)) ([]Result, error) {
	mode := Mode(pieces, budgetChars)
	results := make([]Result, 0, len(pieces))

	for _, w := range buildWindows(pieces, synopsis, meta, budgetChars, mode) {
		for _, idx := range w.members {
			piece := pieces[idx]
			out, err := complete(ctx, c, w.prefix+chunkBlock(piece.Text)+w.instruction)
			if err != nil {
				return nil, fmt.Errorf("contextualize chunk %s: %w", piece.ID, err)
			}
			results = append(results, Result{
				ChunkID:  piece.ID,
				TextHash: piece.TextHash,
				Context:  out,
				Mode:     mode,
			})
			if progress != nil {
				progress(len(results), len(pieces))
			}
		}
	}
	return results, nil
}

type window struct {
	prefix      string
	instruction string
	members     []int
}

func buildWindows(pieces []Piece, synopsis string, meta Meta, budgetChars int, mode string) []window {
	if mode == ModeFull {
		members := make([]int, len(pieces))
		for i := range pieces {
			members[i] = i
		}
		return []window{{
			prefix:      metaBlock(meta) + "<document>\n" + joinPieces(pieces) + "\n</document>\n\n",
			instruction: chunkInstruction,
			members:     members,
		}}
	}

	head := metaBlock(meta) + "<document_summary>\n" + synopsis + "\n</document_summary>\n\n"
	excerptBudget := budgetChars - len(head)
	if excerptBudget < budgetChars/2 {
		excerptBudget = budgetChars / 2
	}
	groupBudget := excerptBudget * 3 / 4

	var windows []window
	start := 0
	for start < len(pieces) {
		end := start + 1
		size := len(pieces[start].Text)
		for end < len(pieces) && size+len(pieces[end].Text) <= groupBudget {
			size += len(pieces[end].Text)
			end++
		}

		lo, hi := start, end
		if lo > 0 && size+len(pieces[lo-1].Text) <= excerptBudget {
			size += len(pieces[lo-1].Text)
			lo--
		}
		if hi < len(pieces) && size+len(pieces[hi].Text) <= excerptBudget {
			hi++
		}

		members := make([]int, 0, end-start)
		for i := start; i < end; i++ {
			members = append(members, i)
		}
		windows = append(windows, window{
			prefix: head +
				"Here is the section of the document surrounding the chunk:\n" +
				"<document_excerpt>\n" + joinPieces(pieces[lo:hi]) + "\n</document_excerpt>\n\n",
			instruction: windowedChunkInstruction,
			members:     members,
		})
		start = end
	}
	return windows
}

func groupByBudget(pieces []Piece, budgetChars int) [][]Piece {
	var groups [][]Piece
	start := 0
	for start < len(pieces) {
		end := start + 1
		size := len(pieces[start].Text)
		for end < len(pieces) && size+len(pieces[end].Text) <= budgetChars {
			size += len(pieces[end].Text)
			end++
		}
		groups = append(groups, pieces[start:end])
		start = end
	}
	return groups
}

// joinPieces concatenates chunk texts, dropping the overlap that chunkers
// repeat at the start of each chunk.
func joinPieces(pieces []Piece) string {
	var b strings.Builder
	prev := ""
	for i, p := range pieces {
		text := p.Text
		if i > 0 {
			if k := overlapLen(prev, text); k > 0 {
				text = text[k:]
			} else {
				b.WriteString(" ")
			}
		}
		b.WriteString(text)
		prev = p.Text
	}
	return b.String()
}

func overlapLen(a, b string) int {
	maxK := min(len(a), len(b), maxOverlapScan)
	for k := maxK; k >= minOverlap; k-- {
		if strings.HasSuffix(a, b[:k]) {
			return k
		}
	}
	return 0
}

func metaBlock(meta Meta) string {
	var lines []string
	if meta.Title != "" {
		lines = append(lines, "Title: "+meta.Title)
	}
	if meta.Source != "" {
		lines = append(lines, "Source: "+meta.Source)
	}
	if meta.Description != "" {
		lines = append(lines, "Description: "+truncate(meta.Description, maxDescriptionChars))
	}
	if len(lines) == 0 {
		return ""
	}
	return "<document_info>\n" + strings.Join(lines, "\n") + "\n</document_info>\n\n"
}

func chunkBlock(text string) string {
	return "Here is the chunk we want to situate within the whole document\n<chunk>\n" + text + "\n</chunk>\n\n"
}

func complete(ctx context.Context, c Completer, user string) (string, error) {
	out, err := c.Complete(ctx, systemPrompt, user)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(thinkBlock.ReplaceAllString(out, "")), nil
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}

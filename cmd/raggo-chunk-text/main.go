package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/jeffrpowell/raggo/pkg/hashing"
	"github.com/jeffrpowell/raggo/pkg/logging"
	"github.com/jeffrpowell/raggo/pkg/schema"
	"github.com/jeffrpowell/raggo/pkg/storage"
)

func main() {
	var (
		documentID   string
		textDir      string
		chunkDir     string
		chunkSize    int
		overlapSize  int
	)

	flag.StringVar(&documentID, "document-id", "", "Document ID")
	flag.StringVar(&textDir, "text-dir", "data/documents/text", "Directory for extracted text")
	flag.StringVar(&chunkDir, "chunk-dir", "data/documents/chunks", "Directory for chunks")
	flag.IntVar(&chunkSize, "chunk-size", 1000, "Chunk size in characters")
	flag.IntVar(&overlapSize, "overlap-size", 100, "Overlap size in characters")
	flag.Parse()

	log := logging.New("raggo-chunk-text")

	if documentID == "" {
		log.Fatal("document-id is required")
	}

	if err := run(log, documentID, textDir, chunkDir, chunkSize, overlapSize); err != nil {
		log.Fatal("Failed: %v", err)
	}
}

func run(log *logging.Logger, documentID, textDir, chunkDir string, chunkSize, overlapSize int) error {
	textPath := filepath.Join(textDir, fmt.Sprintf("%s.json", documentID))
	chunkPath := filepath.Join(chunkDir, fmt.Sprintf("%s.jsonl", documentID))

	if storage.MarkerExists(chunkPath) {
		log.Info("Chunks already exist: %s", chunkPath)
		return nil
	}

	log.Info("Loading extracted text: %s", textPath)

	var extracted schema.ExtractedText
	if err := storage.ReadJSON(textPath, &extracted); err != nil {
		return fmt.Errorf("read extracted text: %w", err)
	}

	log.Info("Chunking text (%d chars, chunk-size=%d, overlap=%d)", 
		extracted.CharCount, chunkSize, overlapSize)

	chunks := createTextChunks(documentID, extracted.Text, chunkSize, overlapSize)

	if err := os.Remove(chunkPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove old chunks: %w", err)
	}

	for _, chunk := range chunks {
		if err := storage.AppendJSONL(chunkPath, chunk); err != nil {
			return fmt.Errorf("append chunk: %w", err)
		}
	}

	log.Info("Wrote %d chunks to: %s", len(chunks), chunkPath)
	return nil
}

func createTextChunks(documentID, text string, chunkSize, overlapSize int) []schema.TextChunk {
	if len(text) == 0 {
		return nil
	}

	text = normalizeWhitespace(text)
	
	var chunks []schema.TextChunk
	chunkIndex := 0
	position := 0

	for position < len(text) {
		end := position + chunkSize
		if end > len(text) {
			end = len(text)
		}

		chunkText := text[position:end]
		
		if position > 0 && position+chunkSize < len(text) {
			chunkText = findSentenceBoundary(text, position, end, chunkText)
			end = position + len(chunkText)
		}

		textHash := hashing.HashString(chunkText)
		chunkID := hashing.HashString(fmt.Sprintf("%s|%d", documentID, chunkIndex))

		chunk := schema.TextChunk{
			ChunkID:    chunkID,
			DocumentID: documentID,
			ChunkIndex: chunkIndex,
			Text:       chunkText,
			TextHash:   textHash,
			CharCount:  len(chunkText),
			CreatedAt:  time.Now(),
		}

		if overlapSize > 0 && position > 0 {
			chunk.OverlapStart = position
			chunk.OverlapEnd = position + overlapSize
		}

		chunks = append(chunks, chunk)
		chunkIndex++
		
		position = end - overlapSize
		if position < 0 {
			position = end
		}
		
		if position >= len(text) {
			break
		}
	}

	return chunks
}

func normalizeWhitespace(text string) string {
	text = strings.ReplaceAll(text, "\r\n", "\n")
	text = strings.ReplaceAll(text, "\r", "\n")
	
	lines := strings.Split(text, "\n")
	var normalized []string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" {
			normalized = append(normalized, trimmed)
		}
	}
	return strings.Join(normalized, " ")
}

func findSentenceBoundary(text string, start, end int, chunkText string) string {
	sentenceEnders := []string{". ", "! ", "? ", ".\n", "!\n", "?\n"}
	
	searchEnd := end
	if searchEnd > len(text) {
		searchEnd = len(text)
	}
	
	searchStart := start
	if searchStart > 0 {
		searchStart = start + len(chunkText) - 200
		if searchStart < start {
			searchStart = start
		}
	}
	
	bestPos := -1
	for i := searchEnd - 1; i >= searchStart && i < len(text); i-- {
		for _, ender := range sentenceEnders {
			if i+len(ender) <= len(text) && text[i:i+len(ender)] == ender {
				bestPos = i + len(ender)
				break
			}
		}
		if bestPos != -1 {
			break
		}
	}
	
	if bestPos > start && bestPos <= end {
		return text[start:bestPos]
	}
	
	return chunkText
}

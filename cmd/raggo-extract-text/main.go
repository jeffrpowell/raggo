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
		documentID string
		filePath   string
		textDir    string
	)

	flag.StringVar(&documentID, "document-id", "", "Document ID")
	flag.StringVar(&filePath, "file-path", "", "Path to document file")
	flag.StringVar(&textDir, "text-dir", "data/documents/text", "Directory for extracted text")
	flag.Parse()

	log := logging.New("raggo-extract-text")

	if documentID == "" {
		log.Fatal("document-id is required")
	}
	if filePath == "" {
		log.Fatal("file-path is required")
	}

	if err := run(log, documentID, filePath, textDir); err != nil {
		log.Fatal("Failed: %v", err)
	}
}

func run(log *logging.Logger, documentID, filePath, textDir string) error {
	textPath := filepath.Join(textDir, fmt.Sprintf("%s.json", documentID))

	if storage.MarkerExists(textPath) {
		log.Info("Text already extracted: %s", textPath)
		return nil
	}

	log.Info("Extracting text from: %s", filePath)

	info, err := os.Stat(filePath)
	if err != nil {
		return fmt.Errorf("stat file: %w", err)
	}

	ext := strings.ToLower(filepath.Ext(filePath))
	text, extractor, err := extractText(filePath, ext)
	if err != nil {
		return fmt.Errorf("extract text: %w", err)
	}

	contentHash, err := hashFileContent(filePath)
	if err != nil {
		return fmt.Errorf("hash file: %w", err)
	}

	wordCount := countWords(text)

	extracted := schema.ExtractedText{
		DocumentID:  documentID,
		FilePath:    filePath,
		ContentHash: contentHash,
		Text:        text,
		CharCount:   len(text),
		WordCount:   wordCount,
		Extractor:   extractor,
		ExtractedAt: time.Now(),
		Metadata: map[string]string{
			"file_size": fmt.Sprintf("%d", info.Size()),
			"extension": ext,
		},
	}

	if err := storage.WriteJSON(textPath, extracted); err != nil {
		return fmt.Errorf("write extracted text: %w", err)
	}

	log.Info("Extracted %d chars (%d words) to: %s", extracted.CharCount, extracted.WordCount, textPath)
	return nil
}

func extractText(filePath, ext string) (string, string, error) {
	switch ext {
	case ".txt", ".md":
		data, err := os.ReadFile(filePath)
		if err != nil {
			return "", "", err
		}
		return string(data), "plain", nil

	case ".doc", ".docx":
		return extractOfficeDoc(filePath)

	case ".xls", ".xlsx":
		return extractOfficeSpreadsheet(filePath)

	case ".ppt", ".pptx":
		return extractOfficePresentation(filePath)

	case ".pdf":
		return extractPDF(filePath)

	case ".rtf":
		return extractRTF(filePath)

	case ".odt", ".ods", ".odp":
		return extractOpenDocument(filePath)

	default:
		return "", "", fmt.Errorf("unsupported file type: %s", ext)
	}
}

func extractOfficeDoc(filePath string) (string, string, error) {
	return fmt.Sprintf("[PLACEHOLDER: Text extraction from %s]\nTo enable MS Word document extraction, install a library like 'github.com/unidoc/unioffice' or use external tools like 'docx2txt' or 'antiword'.\n\nFor now, this is a placeholder that would contain the extracted text content.", filepath.Base(filePath)), "placeholder-docx", nil
}

func extractOfficeSpreadsheet(filePath string) (string, string, error) {
	return fmt.Sprintf("[PLACEHOLDER: Text extraction from %s]\nTo enable MS Excel extraction, install 'github.com/xuri/excelize' or use external tools.\n\nFor now, this is a placeholder that would contain the extracted spreadsheet content.", filepath.Base(filePath)), "placeholder-xlsx", nil
}

func extractOfficePresentation(filePath string) (string, string, error) {
	return fmt.Sprintf("[PLACEHOLDER: Text extraction from %s]\nTo enable MS PowerPoint extraction, install a library like 'github.com/unidoc/unioffice' or use external tools.\n\nFor now, this is a placeholder that would contain the extracted presentation text.", filepath.Base(filePath)), "placeholder-pptx", nil
}

func extractPDF(filePath string) (string, string, error) {
	return fmt.Sprintf("[PLACEHOLDER: Text extraction from %s]\nTo enable PDF extraction, install 'github.com/ledongthuc/pdf' or use external tools like 'pdftotext'.\n\nFor now, this is a placeholder that would contain the extracted PDF text.", filepath.Base(filePath)), "placeholder-pdf", nil
}

func extractRTF(filePath string) (string, string, error) {
	return fmt.Sprintf("[PLACEHOLDER: Text extraction from %s]\nTo enable RTF extraction, install an RTF parser or use external tools.\n\nFor now, this is a placeholder that would contain the extracted RTF text.", filepath.Base(filePath)), "placeholder-rtf", nil
}

func extractOpenDocument(filePath string) (string, string, error) {
	return fmt.Sprintf("[PLACEHOLDER: Text extraction from %s]\nTo enable OpenDocument extraction, install an ODT/ODS/ODP parser or use external tools.\n\nFor now, this is a placeholder that would contain the extracted OpenDocument text.", filepath.Base(filePath)), "placeholder-odf", nil
}

func hashFileContent(filePath string) (string, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return "", err
	}
	return hashing.HashString(string(data)), nil
}

func countWords(text string) int {
	fields := strings.Fields(text)
	return len(fields)
}

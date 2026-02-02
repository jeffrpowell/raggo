package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/jeffrpowell/raggo/pkg/config"
	"github.com/jeffrpowell/raggo/pkg/hashing"
	"github.com/jeffrpowell/raggo/pkg/logging"
	"github.com/jeffrpowell/raggo/pkg/schema"
	"github.com/jeffrpowell/raggo/pkg/storage"
	"github.com/jeffrpowell/raggo/pkg/tika"
	"github.com/jeffrpowell/raggo/pkg/vision"
)

const (
	minCharsForGoodExtraction = 100
	defaultVisionPrompt       = "Extract all text from this image accurately, preserving structure and formatting."
)

type extractConfig struct {
	documentID     string
	filePath       string
	textDir        string
	tikaEndpoint   string
	visionEndpoint string
	visionModel    string
	visionPrompt   string
}

func main() {
	var (
		configPath string
		corpusID   string
	)
	ecfg := extractConfig{}

	flag.StringVar(&configPath, "config", "", "Path to raggo.yml config file")
	flag.StringVar(&corpusID, "corpus-id", "", "Corpus ID from config")
	flag.StringVar(&ecfg.documentID, "document-id", "", "Document ID")
	flag.StringVar(&ecfg.filePath, "file-path", "", "Path to document file")
	flag.StringVar(&ecfg.textDir, "text-dir", "data/documents/text", "Directory for extracted text")
	flag.StringVar(&ecfg.tikaEndpoint, "tika-endpoint", "http://localhost:9998", "Tika HTTP endpoint")
	flag.StringVar(&ecfg.visionEndpoint, "vision-endpoint", "", "OpenAI-compatible vision API endpoint")
	flag.StringVar(&ecfg.visionModel, "vision-model", "gpt-4o", "Vision model name")
	flag.StringVar(&ecfg.visionPrompt, "vision-prompt", defaultVisionPrompt, "Vision extraction prompt")
	flag.Parse()

	log := logging.New("raggo-extract-text")

	var cfg *config.Config
	if configPath != "" {
		var err error
		cfg, err = config.Load(configPath)
		if err != nil {
			log.Fatal("Failed to load config: %v", err)
		}
	}

	ecfg.textDir = config.ResolveDocumentsTextDir(cfg, corpusID, ecfg.textDir)

	if ecfg.documentID == "" {
		log.Fatal("document-id is required")
	}
	if ecfg.filePath == "" {
		log.Fatal("file-path is required")
	}

	if err := run(log, ecfg); err != nil {
		log.Fatal("Failed: %v", err)
	}
}

func run(log *logging.Logger, cfg extractConfig) error {
	textPath := filepath.Join(cfg.textDir, fmt.Sprintf("%s.json", cfg.documentID))

	if storage.MarkerExists(textPath) {
		log.Info("Text already extracted: %s", textPath)
		return nil
	}

	log.Info("Extracting text from: %s", cfg.filePath)

	info, err := os.Stat(cfg.filePath)
	if err != nil {
		return fmt.Errorf("stat file: %w", err)
	}

	ext := strings.ToLower(filepath.Ext(cfg.filePath))
	
	result, err := extractText(log, cfg, ext)
	if err != nil {
		return fmt.Errorf("extract text: %w", err)
	}

	contentHash, err := hashFileContent(cfg.filePath)
	if err != nil {
		return fmt.Errorf("hash file: %w", err)
	}

	wordCount := countWords(result.Text)

	extracted := schema.ExtractedText{
		DocumentID:   cfg.documentID,
		FilePath:     cfg.filePath,
		ContentHash:  contentHash,
		Text:         result.Text,
		CharCount:    len(result.Text),
		WordCount:    wordCount,
		Extractor:    result.Extractor,
		ExtractedAt:  time.Now(),
		TikaMetadata: result.TikaMetadata,
		OCRProvider:  result.OCRProvider,
		ImageCount:   result.ImageCount,
		PageCount:    result.PageCount,
		Metadata: map[string]string{
			"file_size": fmt.Sprintf("%d", info.Size()),
			"extension": ext,
		},
	}

	if err := storage.WriteJSON(textPath, extracted); err != nil {
		return fmt.Errorf("write extracted text: %w", err)
	}

	log.Info("Extracted %d chars (%d words) using %s to: %s", 
		extracted.CharCount, extracted.WordCount, extracted.Extractor, textPath)
	
	if extracted.OCRProvider != "" {
		log.Info("OCR Provider: %s", extracted.OCRProvider)
	}
	if extracted.PageCount > 0 {
		log.Info("Page Count: %d", extracted.PageCount)
	}
	if extracted.ImageCount > 0 {
		log.Info("Image Count: %d", extracted.ImageCount)
	}
	
	return nil
}

type extractionResult struct {
	Text         string
	Extractor    string
	TikaMetadata map[string]interface{}
	OCRProvider  string
	ImageCount   int
	PageCount    int
}

func extractText(log *logging.Logger, cfg extractConfig, ext string) (*extractionResult, error) {
	switch ext {
	case ".txt", ".md":
		data, err := os.ReadFile(cfg.filePath)
		if err != nil {
			return nil, err
		}
		return &extractionResult{
			Text:      string(data),
			Extractor: "plain",
		}, nil

	case ".doc", ".docx", ".xls", ".xlsx", ".ppt", ".pptx", ".pdf", ".rtf", ".odt", ".ods", ".odp":
		return extractWithTika(log, cfg)

	default:
		return nil, fmt.Errorf("unsupported file type: %s", ext)
	}
}

func extractWithTika(log *logging.Logger, cfg extractConfig) (*extractionResult, error) {
	tikaClient := tika.NewClient(cfg.tikaEndpoint)
	
	log.Info("Attempting Tika extraction from: %s", cfg.tikaEndpoint)
	
	tikaResult, err := tikaClient.ExtractText(cfg.filePath)
	if err != nil {
		return nil, fmt.Errorf("tika extraction failed: %w\nEnsure Tika is running at %s", err, cfg.tikaEndpoint)
	}

	result := &extractionResult{
		Text:         strings.TrimSpace(tikaResult.Text),
		Extractor:    "tika",
		TikaMetadata: tikaResult.Metadata,
		PageCount:    tika.ExtractPageCount(tikaResult.Metadata),
		ImageCount:   tika.ExtractImageCount(tikaResult.Metadata),
	}

	log.Info("Tika extracted %d chars", len(result.Text))
	
	needsOCR := shouldUseOCR(log, result, cfg)
	
	if needsOCR {
		log.Info("Text extraction insufficient (%d chars), attempting OCR", len(result.Text))
		
		ext := strings.ToLower(filepath.Ext(cfg.filePath))
		if ext != ".pdf" {
			log.Warn("OCR currently only supported for PDF files, skipping")
			return result, nil
		}
		
		visionClient := vision.NewClient(cfg.visionEndpoint, cfg.visionModel, cfg.visionPrompt)
		
		ocrText, err := visionClient.ExtractTextFromPDF(cfg.filePath)
		if err != nil {
			log.Warn("OCR failed: %v, falling back to Tika-only text", err)
			return result, nil
		}
		
		log.Info("OCR extracted %d chars", len(ocrText))
		
		if len(result.Text) > 0 {
			result.Text = result.Text + "\n\n" + ocrText
			result.OCRProvider = "tika+llm-vision"
		} else {
			result.Text = ocrText
			result.OCRProvider = "llm-vision"
		}
		result.Extractor = "tika+vision-ocr"
	}

	return result, nil
}

func shouldUseOCR(log *logging.Logger, result *extractionResult, cfg extractConfig) bool {
	if cfg.visionEndpoint == "" {
		return false
	}
	
	hasMinimalText := len(result.Text) < minCharsForGoodExtraction
	hasContentToOCR := result.PageCount > 0 || result.ImageCount > 0
	
	if hasMinimalText && hasContentToOCR {
		log.Info("Poor extraction detected: %d chars, %d pages, %d images", 
			len(result.Text), result.PageCount, result.ImageCount)
		return true
	}
	
	return false
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

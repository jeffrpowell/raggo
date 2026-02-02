package main

import (
	"crypto/sha256"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/jeffrpowell/raggo/pkg/config"
	"github.com/jeffrpowell/raggo/pkg/hashing"
	"github.com/jeffrpowell/raggo/pkg/logging"
	"github.com/jeffrpowell/raggo/pkg/schema"
	"github.com/jeffrpowell/raggo/pkg/storage"
)

var supportedExtensions = map[string]string{
	".txt":  "text/plain",
	".md":   "text/markdown",
	".doc":  "application/msword",
	".docx": "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
	".xls":  "application/vnd.ms-excel",
	".xlsx": "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
	".ppt":  "application/vnd.ms-powerpoint",
	".pptx": "application/vnd.openxmlformats-officedocument.presentationml.presentation",
	".pdf":  "application/pdf",
	".rtf":  "application/rtf",
	".odt":  "application/vnd.oasis.opendocument.text",
	".ods":  "application/vnd.oasis.opendocument.spreadsheet",
	".odp":  "application/vnd.oasis.opendocument.presentation",
}

func main() {
	var (
		configPath  string
		corpusID    string
		rootPath    string
		manifestDir string
	)

	flag.StringVar(&configPath, "config", "", "Path to raggo.yml config file")
	flag.StringVar(&corpusID, "corpus-id", "", "Corpus ID from config")
	flag.StringVar(&rootPath, "root-path", "", "Root path to scan recursively")
	flag.StringVar(&manifestDir, "manifest-dir", "data/manifests", "Directory for manifest files")
	flag.Parse()

	log := logging.New("raggo-scan-documents")

	var cfg *config.Config
	if configPath != "" {
		var err error
		cfg, err = config.Load(configPath)
		if err != nil {
			log.Fatal("Failed to load config: %v", err)
		}
	}

	rootPath = config.ResolveDocumentsSourcesDir(cfg, corpusID, rootPath)
	manifestDir = config.ResolveDocumentsTextDir(cfg, corpusID, manifestDir)

	if rootPath == "" {
		log.Fatal("root-path is required")
	}

	if err := run(log, rootPath, manifestDir); err != nil {
		log.Fatal("Failed: %v", err)
	}
}

func run(log *logging.Logger, rootPath, manifestDir string) error {
	absRootPath, err := filepath.Abs(rootPath)
	if err != nil {
		return fmt.Errorf("resolve root path: %w", err)
	}

	info, err := os.Stat(absRootPath)
	if err != nil {
		return fmt.Errorf("stat root path: %w", err)
	}
	if !info.IsDir() {
		return fmt.Errorf("root path is not a directory: %s", absRootPath)
	}

	log.Info("Scanning directory: %s", absRootPath)

	rootHash := hashing.HashString(absRootPath)
	manifestPath := filepath.Join(manifestDir, fmt.Sprintf("%s.jsonl", rootHash))

	if err := os.Remove(manifestPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove old manifest: %w", err)
	}

	count := 0
	err = filepath.Walk(absRootPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			log.Warn("Error accessing path %s: %v", path, err)
			return nil
		}

		if info.IsDir() {
			return nil
		}

		ext := strings.ToLower(filepath.Ext(path))
		fileType, supported := supportedExtensions[ext]
		if !supported {
			return nil
		}

		relPath, err := filepath.Rel(absRootPath, path)
		if err != nil {
			log.Warn("Error getting relative path for %s: %v", path, err)
			return nil
		}

		contentHash, err := hashFile(path)
		if err != nil {
			log.Warn("Error hashing file %s: %v", path, err)
			return nil
		}

		documentID := hashing.HashString(fmt.Sprintf("%s|%s", absRootPath, relPath))

		manifest := schema.DocumentManifest{
			DocumentID:   documentID,
			RootPath:     absRootPath,
			RelativePath: relPath,
			AbsolutePath: path,
			FileName:     filepath.Base(path),
			FileSize:     info.Size(),
			FileType:     fileType,
			Extension:    ext,
			ModifiedAt:   info.ModTime(),
			ScannedAt:    time.Now(),
			ContentHash:  contentHash,
		}

		if err := storage.AppendJSONL(manifestPath, manifest); err != nil {
			return fmt.Errorf("append to manifest: %w", err)
		}

		count++
		return nil
	})

	if err != nil {
		return fmt.Errorf("walk directory: %w", err)
	}

	log.Info("Scanned %d documents, wrote manifest: %s", count, manifestPath)
	return nil
}

func hashFile(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}

	return fmt.Sprintf("%x", h.Sum(nil)), nil
}

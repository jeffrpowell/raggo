package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/jeffrpowell/raggo/pkg/config"
	"github.com/jeffrpowell/raggo/pkg/hashing"
	"github.com/jeffrpowell/raggo/pkg/logging"
	"github.com/jeffrpowell/raggo/pkg/schema"
	"github.com/jeffrpowell/raggo/pkg/storage"
	"github.com/mmcdole/gofeed"
)

func main() {
	var (
		configPath  string
		corpusID    string
		feedURL     string
		rawDir      string
		manifestDir string
	)

	flag.StringVar(&configPath, "config", "", "Path to raggo.yml config file")
	flag.StringVar(&corpusID, "corpus-id", "", "Corpus ID from config")
	flag.StringVar(&feedURL, "feed-url", "", "RSS feed URL")
	flag.StringVar(&rawDir, "raw-dir", "data/raw", "Directory for raw feed files")
	flag.StringVar(&manifestDir, "manifest-dir", "data/manifests", "Directory for manifest files")
	flag.Parse()

	log := logging.New("raggo-rss-podcast")

	var cfg *config.Config
	if configPath != "" {
		var err error
		cfg, err = config.Load(configPath)
		if err != nil {
			log.Fatal("Failed to load config: %v", err)
		}
	}

	rawDir = config.ResolveAudioDir(cfg, corpusID, rawDir)
	manifestDir = config.ResolveAudioDir(cfg, corpusID, manifestDir)

	if feedURL == "" {
		log.Fatal("feed-url is required")
	}

	if err := run(log, feedURL, rawDir, manifestDir); err != nil {
		log.Fatal("Failed: %v", err)
	}
}

func run(log *logging.Logger, feedURL, rawDir, manifestDir string) error {
	log.Info("Fetching RSS feed: %s", feedURL)

	fp := gofeed.NewParser()
	feed, err := fp.ParseURL(feedURL)
	if err != nil {
		return fmt.Errorf("parse feed: %w", err)
	}

	log.Info("Parsed feed: %s (%d episodes)", feed.Title, len(feed.Items))

	feedHash := hashing.HashString(feedURL)
	rawPath := filepath.Join(rawDir, fmt.Sprintf("%s.xml", feedHash))
	
	if err := downloadRawFeed(feedURL, rawPath); err != nil {
		return fmt.Errorf("download raw feed: %w", err)
	}
	log.Info("Saved raw feed to: %s", rawPath)

	podcastFeed := schema.PodcastFeed{
		Title:       feed.Title,
		Description: feed.Description,
		Link:        feed.Link,
		Language:    feed.Language,
		Author:      getAuthor(feed),
		FeedURL:     feedURL,
		ParsedAt:    time.Now(),
		FeedHash:    feedHash,
	}

	feedMetaPath := filepath.Join(rawDir, fmt.Sprintf("%s.json", feedHash))
	if err := storage.WriteJSON(feedMetaPath, podcastFeed); err != nil {
		return fmt.Errorf("write feed metadata: %w", err)
	}

	manifestPath := filepath.Join(manifestDir, fmt.Sprintf("%s.jsonl", feedHash))
	
	if err := os.Remove(manifestPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove old manifest: %w", err)
	}

	for _, item := range feed.Items {
		audioURL := getAudioURL(item)
		if audioURL == "" {
			log.Warn("Skipping episode without audio: %s", item.Title)
			continue
		}

		episodeID := hashing.HashString(fmt.Sprintf("%s|%s", feedURL, item.GUID))
		
		manifest := schema.EpisodeManifest{
			EpisodeID:   episodeID,
			FeedURL:     feedURL,
			Title:       item.Title,
			Description: item.Description,
			AudioURL:    audioURL,
			PublishedAt: getPubDate(item),
			Duration:    getDuration(item),
			GUID:        item.GUID,
		}

		manifestHash, err := hashing.HashStruct(manifest)
		if err != nil {
			return fmt.Errorf("hash manifest: %w", err)
		}
		manifest.MetadataHash = manifestHash

		if err := storage.AppendJSONL(manifestPath, manifest); err != nil {
			return fmt.Errorf("append to manifest: %w", err)
		}
	}

	log.Info("Wrote %d episodes to manifest: %s", len(feed.Items), manifestPath)
	return nil
}

func downloadRawFeed(url, path string) (err error) {
	fp := gofeed.NewParser()
	
	resp, err := fp.Client.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if err := storage.EnsureDir(filepath.Dir(path)); err != nil {
		return err
	}

	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := f.Close(); closeErr != nil && err == nil {
			err = closeErr
		}
	}()

	_, err = io.Copy(f, resp.Body)
	return err
}

func getAuthor(feed *gofeed.Feed) string {
	if feed.Author != nil {
		return feed.Author.Name
	}
	if feed.ITunesExt != nil {
		return feed.ITunesExt.Author
	}
	return ""
}

func getAudioURL(item *gofeed.Item) string {
	for _, enc := range item.Enclosures {
		if enc.Type != "" && (enc.Type[:5] == "audio" || enc.Type[:9] == "application") {
			return enc.URL
		}
	}
	return ""
}

func getPubDate(item *gofeed.Item) time.Time {
	if item.PublishedParsed != nil {
		return *item.PublishedParsed
	}
	return time.Now()
}

func getDuration(item *gofeed.Item) int {
	if item.ITunesExt != nil && item.ITunesExt.Duration != "" {
		return 0
	}
	return 0
}

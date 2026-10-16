package main

import (
	"context"
	"flag"
	"fmt"

	"github.com/jeffrpowell/raggo/pkg/config"
	"github.com/jeffrpowell/raggo/pkg/logging"
	"github.com/jeffrpowell/raggo/pkg/vectorstore"
	qdrant "github.com/qdrant/go-client/qdrant"
)

func main() {
	var (
		configPath  string
		qdrantHost  string
		qdrantPort  int
		collection  string
		description string
	)

	flag.StringVar(&configPath, "config", "", "Path to raggo.yml config file")
	flag.StringVar(&qdrantHost, "qdrant-host", "", "Qdrant host (default: config qdrant.host)")
	flag.IntVar(&qdrantPort, "qdrant-port", 0, "Qdrant gRPC port (default: config qdrant.port)")
	flag.StringVar(&collection, "collection", "", "Qdrant collection name")
	flag.StringVar(&description, "description", "", "Summary of the collection's contents, stored in its metadata")
	flag.Parse()

	log := logging.New("raggo-describe-collection")

	var cfg *config.Config
	if configPath != "" {
		var err error
		cfg, err = config.Load(configPath)
		if err != nil {
			log.Fatal("Failed to load config: %v", err)
		}
	}

	qdrantHost = config.ResolveQdrantHost(cfg, qdrantHost)
	qdrantPort = config.ResolveQdrantPort(cfg, qdrantPort)
	qdrantAPIKey := config.ResolveQdrantAPIKey(cfg)

	if collection == "" || description == "" {
		log.Fatal("collection and description are required")
	}

	if err := run(log, qdrantHost, qdrantPort, qdrantAPIKey, collection, description); err != nil {
		log.Fatal("Failed: %v", err)
	}
}

func run(log *logging.Logger, qdrantHost string, qdrantPort int, qdrantAPIKey, collection, description string) error {
	client, err := qdrant.NewClient(&qdrant.Config{
		Host:   qdrantHost,
		Port:   qdrantPort,
		APIKey: qdrantAPIKey,
	})
	if err != nil {
		return fmt.Errorf("connect to qdrant: %w", err)
	}
	defer client.Close()

	exists, changed, err := vectorstore.SetDescription(context.Background(), client, collection, description)
	if err != nil {
		return err
	}
	switch {
	case !exists:
		log.Warn("Collection %s doesn't exist yet; its description will be set on a run after something is indexed", collection)
	case changed:
		log.Info("Updated description of collection %s", collection)
	default:
		log.Info("Description of collection %s is up to date", collection)
	}
	return nil
}

.PHONY: podcast text filesystem build install-deps clean help
.DEFAULT_GOAL := help

BIN_DIR := bin

help:
	@echo "Raggo: Deterministic RAG Ingestion Pipeline"
	@echo ""
	@echo "Pipelines:"
	@echo "  make podcast FEED_URL=<url>  - Run podcast ingestion pipeline"
	@echo "  make text                    - Run text ingestion pipeline (future)"
	@echo "  make filesystem              - Run filesystem pipeline (future)"
	@echo ""
	@echo "Development:"
	@echo "  make build                   - Build all binaries"
	@echo "  make install-deps            - Install Go dependencies"
	@echo "  make clean                   - Remove all artifacts"
	@echo ""
	@echo "See pipelines/<type>/Makefile for pipeline-specific options"

podcast:
	$(MAKE) -C pipelines/podcast

text:
	@echo "Text pipeline not yet implemented"
	@exit 1

filesystem:
	@echo "Filesystem pipeline not yet implemented"
	@exit 1

build: $(BIN_DIR)/raggo-rss-podcast \
       $(BIN_DIR)/raggo-download-audio \
       $(BIN_DIR)/raggo-stt-audio \
       $(BIN_DIR)/raggo-normalize-podcast \
       $(BIN_DIR)/raggo-chunk-podcast \
       $(BIN_DIR)/raggo-embed-podcast \
       $(BIN_DIR)/raggo-index-podcast

$(BIN_DIR)/%: cmd/%/main.go pkg/**/*.go
	@mkdir -p $(BIN_DIR)
	go build -o $@ ./cmd/$*

install-deps:
	go mod download
	go mod tidy

clean:
	rm -rf data/*
	$(MAKE) -C pipelines/podcast clean 2>/dev/null || true

clean-all: clean
	rm -rf $(BIN_DIR)

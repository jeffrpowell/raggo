.PHONY: all clean help scan extract chunks embeddings index
.DEFAULT_GOAL := help

ROOT_DIR := $(shell cd ../.. && pwd)
BIN_DIR := $(ROOT_DIR)/bin
DATA_DIR := $(ROOT_DIR)/data
DOCUMENT_DATA := $(DATA_DIR)/documents

MANIFEST_DIR := $(DATA_DIR)/manifests
TEXT_DIR := $(DOCUMENT_DATA)/text
CHUNK_DIR := $(DOCUMENT_DATA)/chunks
EMBEDDING_DIR := $(DOCUMENT_DATA)/embeddings
INDEX_DIR := $(DOCUMENT_DATA)/index

ROOT_PATH ?=
ROOT_HASH := $(shell echo -n "$(ROOT_PATH)" | sha256sum | cut -d' ' -f1)
MANIFEST := $(MANIFEST_DIR)/$(ROOT_HASH).jsonl

EMBED_ENDPOINT ?=
QDRANT_HOST ?= localhost
QDRANT_PORT ?= 6334
COLLECTION ?= raggo
CORPUS ?= default
CHUNK_SIZE ?= 1000
OVERLAP_SIZE ?= 100

help:
	@echo "Document Pipeline"
	@echo ""
	@echo "Usage:"
	@echo "  make all ROOT_PATH=<path>         - Run full pipeline"
	@echo "  make scan ROOT_PATH=<path>        - Scan directory for documents"
	@echo "  make extract ROOT_PATH=<path>     - Extract text from documents"
	@echo "  make chunks ROOT_PATH=<path>      - Chunk extracted text"
	@echo "  make embeddings ROOT_PATH=<path>  - Generate embeddings"
	@echo "  make index ROOT_PATH=<path>       - Index into Qdrant"
	@echo "  make clean                        - Remove artifacts"
	@echo ""
	@echo "Configuration:"
	@echo "  ROOT_PATH       - Root directory to scan (required)"
	@echo "  EMBED_ENDPOINT  - Embedding HTTP endpoint"
	@echo "  QDRANT_HOST     - Qdrant host (default: localhost)"
	@echo "  QDRANT_PORT     - Qdrant port (default: 6334)"
	@echo "  CHUNK_SIZE      - Chunk size in chars (default: 1000)"
	@echo "  OVERLAP_SIZE    - Overlap size in chars (default: 100)"

all: $(INDEX_DIR)/qdrant.done

scan: $(MANIFEST)

$(MANIFEST): $(BIN_DIR)/raggo-scan-documents
	@if [ -z "$(ROOT_PATH)" ]; then \
		echo "Error: ROOT_PATH is required"; \
		exit 1; \
	fi
	@mkdir -p $(MANIFEST_DIR)
	$(BIN_DIR)/raggo-scan-documents \
		-root-path="$(ROOT_PATH)" \
		-manifest-dir="$(MANIFEST_DIR)"

DOCUMENT_IDS := $(shell if [ -f "$(MANIFEST)" ]; then \
	jq -r '.document_id' "$(MANIFEST)"; \
fi)

TEXT_FILES := $(foreach id,$(DOCUMENT_IDS),$(TEXT_DIR)/$(id).json)
CHUNK_FILES := $(foreach id,$(DOCUMENT_IDS),$(CHUNK_DIR)/$(id).jsonl)
EMBEDDING_FILES := $(foreach id,$(DOCUMENT_IDS),$(EMBEDDING_DIR)/$(id).jsonl)
INDEX_FILES := $(foreach id,$(DOCUMENT_IDS),$(INDEX_DIR)/$(id).done)

extract: $(MANIFEST) $(TEXT_FILES)

$(TEXT_DIR)/%.json: $(BIN_DIR)/raggo-extract-text $(MANIFEST)
	@mkdir -p $(TEXT_DIR)
	@file_path=$$(jq -r "select(.document_id == \"$*\") | .absolute_path" "$(MANIFEST)" | head -1); \
	if [ -z "$$file_path" ]; then \
		echo "Error: Document $* not found in manifest"; \
		exit 1; \
	fi; \
	$(BIN_DIR)/raggo-extract-text \
		-document-id="$*" \
		-file-path="$$file_path" \
		-text-dir="$(TEXT_DIR)"

chunks: $(CHUNK_FILES)

$(CHUNK_DIR)/%.jsonl: $(TEXT_DIR)/%.json $(BIN_DIR)/raggo-chunk-text
	@mkdir -p $(CHUNK_DIR)
	$(BIN_DIR)/raggo-chunk-text \
		-document-id="$*" \
		-text-dir="$(TEXT_DIR)" \
		-chunk-dir="$(CHUNK_DIR)" \
		-chunk-size=$(CHUNK_SIZE) \
		-overlap-size=$(OVERLAP_SIZE)

embeddings: $(EMBEDDING_FILES)

$(EMBEDDING_DIR)/%.jsonl: $(CHUNK_DIR)/%.jsonl $(BIN_DIR)/raggo-embed-text
	@mkdir -p $(EMBEDDING_DIR)
	$(BIN_DIR)/raggo-embed-text \
		-document-id="$*" \
		-chunk-dir="$(CHUNK_DIR)" \
		-embedding-dir="$(EMBEDDING_DIR)" \
		$(if $(EMBED_ENDPOINT),-embed-endpoint="$(EMBED_ENDPOINT)",)

index: $(INDEX_FILES)

$(INDEX_DIR)/%.done: $(EMBEDDING_DIR)/%.jsonl $(BIN_DIR)/raggo-index-documents
	@mkdir -p $(INDEX_DIR)
	$(BIN_DIR)/raggo-index-documents \
		-document-id="$*" \
		-manifest-dir="$(MANIFEST_DIR)" \
		-chunk-dir="$(CHUNK_DIR)" \
		-embedding-dir="$(EMBEDDING_DIR)" \
		-index-dir="$(INDEX_DIR)" \
		-qdrant-host="$(QDRANT_HOST)" \
		-qdrant-port=$(QDRANT_PORT) \
		-collection="$(COLLECTION)" \
		-corpus="$(CORPUS)"

$(INDEX_DIR)/qdrant.done: $(INDEX_FILES)
	@echo "All documents indexed successfully"
	@touch $@

clean:
	rm -rf $(DOCUMENT_DATA) $(MANIFEST_DIR)/$(ROOT_HASH).jsonl

.PRECIOUS: $(TEXT_DIR)/%.json \
           $(CHUNK_DIR)/%.jsonl \
           $(EMBEDDING_DIR)/%.jsonl \
           $(INDEX_DIR)/%.done

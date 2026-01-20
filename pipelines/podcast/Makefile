.PHONY: all clean help rss audio transcripts normalize chunks embeddings index
.DEFAULT_GOAL := help

ROOT_DIR := $(shell cd ../.. && pwd)
BIN_DIR := $(ROOT_DIR)/bin
DATA_DIR := $(ROOT_DIR)/data
PODCAST_DATA := $(DATA_DIR)/podcast

RAW_DIR := $(PODCAST_DATA)/raw
MANIFEST_DIR := $(PODCAST_DATA)/manifests
AUDIO_DIR := $(PODCAST_DATA)/audio
TRANSCRIPT_DIR := $(PODCAST_DATA)/transcripts
CHUNK_DIR := $(PODCAST_DATA)/chunks
EMBEDDING_DIR := $(PODCAST_DATA)/embeddings
INDEX_DIR := $(PODCAST_DATA)/index

FEED_URL ?=
FEED_HASH := $(shell echo -n "$(FEED_URL)" | sha256sum | cut -d' ' -f1)
MANIFEST := $(MANIFEST_DIR)/$(FEED_HASH).jsonl

STT_ENDPOINT ?=
EMBED_ENDPOINT ?=
QDRANT_HOST ?= localhost
QDRANT_PORT ?= 6334
COLLECTION ?= raggo
CORPUS ?= default
WORKERS ?= 4

help:
	@echo "Podcast Pipeline"
	@echo ""
	@echo "Usage:"
	@echo "  make all FEED_URL=<url>     - Run full pipeline"
	@echo "  make rss FEED_URL=<url>     - Ingest RSS feed"
	@echo "  make audio FEED_URL=<url>   - Download audio"
	@echo "  make clean                  - Remove artifacts"
	@echo ""
	@echo "Configuration:"
	@echo "  FEED_URL        - RSS feed URL (required)"
	@echo "  STT_ENDPOINT    - Speech-to-text HTTP endpoint"
	@echo "  EMBED_ENDPOINT  - Embedding HTTP endpoint"
	@echo "  QDRANT_HOST     - Qdrant host (default: localhost)"
	@echo "  QDRANT_PORT     - Qdrant port (default: 6334)"
	@echo "  WORKERS         - Concurrent workers (default: 4)"

all: $(INDEX_DIR)/qdrant.done

rss: $(MANIFEST)

$(MANIFEST): $(BIN_DIR)/raggo-rss-podcast
	@if [ -z "$(FEED_URL)" ]; then \
		echo "Error: FEED_URL is required"; \
		exit 1; \
	fi
	@mkdir -p $(RAW_DIR) $(MANIFEST_DIR)
	$(BIN_DIR)/raggo-rss-podcast \
		-feed-url="$(FEED_URL)" \
		-raw-dir="$(RAW_DIR)" \
		-manifest-dir="$(MANIFEST_DIR)"

EPISODE_IDS := $(shell if [ -f "$(MANIFEST)" ]; then \
	jq -r '.episode_id' "$(MANIFEST)"; \
fi)

AUDIO_FILES := $(foreach id,$(EPISODE_IDS),$(AUDIO_DIR)/$(id).done)
TRANSCRIPT_FILES := $(foreach id,$(EPISODE_IDS),$(TRANSCRIPT_DIR)/$(id).json)
NORMALIZED_FILES := $(foreach id,$(EPISODE_IDS),$(TRANSCRIPT_DIR)/$(id).normalized.json)
CHUNK_FILES := $(foreach id,$(EPISODE_IDS),$(CHUNK_DIR)/$(id).jsonl)
EMBEDDING_FILES := $(foreach id,$(EPISODE_IDS),$(EMBEDDING_DIR)/$(id).jsonl)
INDEX_FILES := $(foreach id,$(EPISODE_IDS),$(INDEX_DIR)/$(id).done)

audio: $(MANIFEST) $(AUDIO_FILES)

$(AUDIO_DIR)/%.done: $(BIN_DIR)/raggo-download-audio $(MANIFEST)
	@mkdir -p $(AUDIO_DIR)
	@audio_file=$$(jq -r "select(.episode_id == \"$*\") | .audio_url" "$(MANIFEST)" | head -1); \
	if [ -z "$$audio_file" ]; then \
		echo "Error: Episode $* not found in manifest"; \
		exit 1; \
	fi; \
	episode_manifest=$$(mktemp); \
	jq -c "select(.episode_id == \"$*\")" "$(MANIFEST)" > $$episode_manifest; \
	$(BIN_DIR)/raggo-download-audio \
		-manifest="$$episode_manifest" \
		-audio-dir="$(AUDIO_DIR)" \
		-workers=1; \
	rm $$episode_manifest
	@touch $@

transcripts: $(TRANSCRIPT_FILES)

$(TRANSCRIPT_DIR)/%.json: $(AUDIO_DIR)/%.done $(BIN_DIR)/raggo-stt-audio
	@mkdir -p $(TRANSCRIPT_DIR)
	@audio_file=$$(find $(AUDIO_DIR) -name "$*.mp3" -o -name "$*.m4a" -o -name "$*.wav" | head -n1); \
	if [ -z "$$audio_file" ]; then \
		echo "Error: Audio file not found for episode $*"; \
		exit 1; \
	fi; \
	$(BIN_DIR)/raggo-stt-audio \
		-episode-id="$*" \
		-audio-path="$$audio_file" \
		-transcript-dir="$(TRANSCRIPT_DIR)" \
		$(if $(STT_ENDPOINT),-stt-endpoint="$(STT_ENDPOINT)",)

normalize: $(NORMALIZED_FILES)

$(TRANSCRIPT_DIR)/%.normalized.json: $(TRANSCRIPT_DIR)/%.json $(BIN_DIR)/raggo-normalize-podcast
	$(BIN_DIR)/raggo-normalize-podcast \
		-episode-id="$*" \
		-transcript-dir="$(TRANSCRIPT_DIR)"

chunks: $(CHUNK_FILES)

$(CHUNK_DIR)/%.jsonl: $(TRANSCRIPT_DIR)/%.normalized.json $(BIN_DIR)/raggo-chunk-podcast
	@mkdir -p $(CHUNK_DIR)
	$(BIN_DIR)/raggo-chunk-podcast \
		-episode-id="$*" \
		-transcript-dir="$(TRANSCRIPT_DIR)" \
		-chunk-dir="$(CHUNK_DIR)"

embeddings: $(EMBEDDING_FILES)

$(EMBEDDING_DIR)/%.jsonl: $(CHUNK_DIR)/%.jsonl $(BIN_DIR)/raggo-embed-podcast
	@mkdir -p $(EMBEDDING_DIR)
	$(BIN_DIR)/raggo-embed-podcast \
		-episode-id="$*" \
		-chunk-dir="$(CHUNK_DIR)" \
		-embedding-dir="$(EMBEDDING_DIR)" \
		$(if $(EMBED_ENDPOINT),-embed-endpoint="$(EMBED_ENDPOINT)",)

index: $(INDEX_FILES)

$(INDEX_DIR)/%.done: $(EMBEDDING_DIR)/%.jsonl $(BIN_DIR)/raggo-index-podcast
	@mkdir -p $(INDEX_DIR)
	$(BIN_DIR)/raggo-index-podcast \
		-episode-id="$*" \
		-chunk-dir="$(CHUNK_DIR)" \
		-embedding-dir="$(EMBEDDING_DIR)" \
		-index-dir="$(INDEX_DIR)" \
		-qdrant-host="$(QDRANT_HOST)" \
		-qdrant-port=$(QDRANT_PORT) \
		-collection="$(COLLECTION)" \
		-corpus="$(CORPUS)"

$(INDEX_DIR)/qdrant.done: $(INDEX_FILES)
	@echo "All episodes indexed successfully"
	@touch $@

clean:
	rm -rf $(PODCAST_DATA)

.PRECIOUS: $(TRANSCRIPT_DIR)/%.json \
           $(TRANSCRIPT_DIR)/%.normalized.json \
           $(CHUNK_DIR)/%.jsonl \
           $(EMBEDDING_DIR)/%.jsonl \
           $(INDEX_DIR)/%.done \
           $(AUDIO_DIR)/%.done

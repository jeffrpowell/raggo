# Raggo Pipeline Configuration Example
# Copy this file to config.mk and customize as needed
# Then include it in your make commands: make -f config.mk pipeline

FEED_URL = https://feeds.example.com/podcast.rss

STT_ENDPOINT = http://localhost:8000/stt

EMBED_ENDPOINT = http://localhost:8001/embed

QDRANT_HOST = localhost
QDRANT_PORT = 6334

COLLECTION = my-podcast-collection
CORPUS = my-corpus

WORKERS = 8

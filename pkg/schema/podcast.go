package schema

import "time"

type PodcastFeed struct {
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Link        string    `json:"link"`
	Language    string    `json:"language"`
	Author      string    `json:"author"`
	FeedURL     string    `json:"feed_url"`
	ParsedAt    time.Time `json:"parsed_at"`
	FeedHash    string    `json:"feed_hash"`
}

type EpisodeManifest struct {
	EpisodeID   string    `json:"episode_id"`
	FeedURL     string    `json:"feed_url"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	AudioURL    string    `json:"audio_url"`
	PublishedAt time.Time `json:"published_at"`
	Duration    int       `json:"duration"`
	GUID        string    `json:"guid"`
	MetadataHash string   `json:"metadata_hash"`
}

type AudioMetadata struct {
	EpisodeID   string    `json:"episode_id"`
	FilePath    string    `json:"file_path"`
	FileSize    int64     `json:"file_size"`
	ContentHash string    `json:"content_hash"`
	MimeType    string    `json:"mime_type"`
	Duration    float64   `json:"duration"`
	DownloadedAt time.Time `json:"downloaded_at"`
}

type TranscriptSegment struct {
	Start   float64 `json:"start"`
	End     float64 `json:"end"`
	Text    string  `json:"text"`
	Speaker string  `json:"speaker,omitempty"`
}

type RawTranscript struct {
	EpisodeID    string               `json:"episode_id"`
	AudioHash    string               `json:"audio_hash"`
	Model        string               `json:"model"`
	ModelVersion string               `json:"model_version"`
	Segments     []TranscriptSegment  `json:"segments"`
	GeneratedAt  time.Time            `json:"generated_at"`
	TranscriptHash string             `json:"transcript_hash"`
}

type NormalizedTranscript struct {
	EpisodeID      string               `json:"episode_id"`
	AudioHash      string               `json:"audio_hash"`
	RawTranscriptHash string            `json:"raw_transcript_hash"`
	Segments       []TranscriptSegment  `json:"segments"`
	NormalizedAt   time.Time            `json:"normalized_at"`
	NormalizedHash string               `json:"normalized_hash"`
}

type Chunk struct {
	ChunkID      string    `json:"chunk_id"`
	EpisodeID    string    `json:"episode_id"`
	ChunkIndex   int       `json:"chunk_index"`
	StartTime    float64   `json:"start_time"`
	EndTime      float64   `json:"end_time"`
	Text         string    `json:"text"`
	TextHash     string    `json:"text_hash"`
	OverlapStart float64   `json:"overlap_start,omitempty"`
	OverlapEnd   float64   `json:"overlap_end,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
}

type Embedding struct {
	ChunkID      string    `json:"chunk_id"`
	EpisodeID    string    `json:"episode_id"`
	ChunkHash    string    `json:"chunk_hash"`
	ContextHash  string    `json:"context_hash,omitempty"`
	Model        string    `json:"model"`
	ModelVersion string    `json:"model_version"`
	Vector       []float32 `json:"vector"`
	Dimension    int       `json:"dimension"`
	GeneratedAt  time.Time `json:"generated_at"`
}

type IndexPayload struct {
	PointID      string            `json:"point_id"`
	ChunkID      string            `json:"chunk_id"`
	EpisodeID    string            `json:"episode_id"`
	Corpus       string            `json:"corpus"`
	SourceType   string            `json:"source_type"`
	SourceID     string            `json:"source_id"`
	StartTime    float64           `json:"start_time"`
	EndTime      float64           `json:"end_time"`
	Text         string            `json:"text"`
	Context      string            `json:"context,omitempty"`
	Model        string            `json:"model"`
	ModelVersion string            `json:"model_version"`
	Metadata     map[string]string `json:"metadata,omitempty"`
	IndexedAt    time.Time         `json:"indexed_at"`
}

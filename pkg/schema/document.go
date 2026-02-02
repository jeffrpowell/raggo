package schema

import "time"

type DocumentManifest struct {
	DocumentID   string            `json:"document_id"`
	RootPath     string            `json:"root_path"`
	RelativePath string            `json:"relative_path"`
	AbsolutePath string            `json:"absolute_path"`
	FileName     string            `json:"file_name"`
	FileSize     int64             `json:"file_size"`
	FileType     string            `json:"file_type"`
	Extension    string            `json:"extension"`
	ModifiedAt   time.Time         `json:"modified_at"`
	ScannedAt    time.Time         `json:"scanned_at"`
	ContentHash  string            `json:"content_hash"`
	Metadata     map[string]string `json:"metadata,omitempty"`
}

type ExtractedText struct {
	DocumentID    string                 `json:"document_id"`
	FilePath      string                 `json:"file_path"`
	ContentHash   string                 `json:"content_hash"`
	Text          string                 `json:"text"`
	CharCount     int                    `json:"char_count"`
	WordCount     int                    `json:"word_count"`
	Extractor     string                 `json:"extractor"`
	ExtractedAt   time.Time              `json:"extracted_at"`
	Metadata      map[string]string      `json:"metadata,omitempty"`
	TikaMetadata  map[string]interface{} `json:"tika_metadata,omitempty"`
	OCRProvider   string                 `json:"ocr_provider,omitempty"`
	ImageCount    int                    `json:"image_count,omitempty"`
	PageCount     int                    `json:"page_count,omitempty"`
}

type TextChunk struct {
	ChunkID      string            `json:"chunk_id"`
	DocumentID   string            `json:"document_id"`
	ChunkIndex   int               `json:"chunk_index"`
	Text         string            `json:"text"`
	TextHash     string            `json:"text_hash"`
	CharCount    int               `json:"char_count"`
	OverlapStart int               `json:"overlap_start,omitempty"`
	OverlapEnd   int               `json:"overlap_end,omitempty"`
	CreatedAt    time.Time         `json:"created_at"`
	Metadata     map[string]string `json:"metadata,omitempty"`
}

type TextEmbedding struct {
	ChunkID      string    `json:"chunk_id"`
	DocumentID   string    `json:"document_id"`
	ChunkHash    string    `json:"chunk_hash"`
	Model        string    `json:"model"`
	ModelVersion string    `json:"model_version"`
	Vector       []float32 `json:"vector"`
	Dimension    int       `json:"dimension"`
	GeneratedAt  time.Time `json:"generated_at"`
}

type DocumentIndexPayload struct {
	PointID      string            `json:"point_id"`
	ChunkID      string            `json:"chunk_id"`
	DocumentID   string            `json:"document_id"`
	Corpus       string            `json:"corpus"`
	SourceType   string            `json:"source_type"`
	SourcePath   string            `json:"source_path"`
	FileName     string            `json:"file_name"`
	FileType     string            `json:"file_type"`
	Text         string            `json:"text"`
	Model        string            `json:"model"`
	ModelVersion string            `json:"model_version"`
	Metadata     map[string]string `json:"metadata,omitempty"`
	IndexedAt    time.Time         `json:"indexed_at"`
}

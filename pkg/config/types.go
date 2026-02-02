package config

type Config struct {
	Storage      StorageConfig      `yaml:"storage"`
	Qdrant       QdrantConfig       `yaml:"qdrant"`
	Services     ServicesConfig     `yaml:"services"`
	Orchestrator OrchestratorConfig `yaml:"orchestrator"`
	Corpora      []CorpusConfig     `yaml:"corpora"`
}

type StorageConfig struct {
	Root                string                `yaml:"root"`
	RetainIntermediates bool                  `yaml:"retain_intermediates"`
	Podcast             PodcastStorageConfig  `yaml:"podcast"`
	Documents           DocumentStorageConfig `yaml:"documents"`
}

type PodcastStorageConfig struct {
	Audio       string `yaml:"audio"`
	Transcripts string `yaml:"transcripts"`
	Chunks      string `yaml:"chunks"`
	Embeddings  string `yaml:"embeddings"`
	Index       string `yaml:"index"`
}

type DocumentStorageConfig struct {
	Sources    string `yaml:"sources"`
	Text       string `yaml:"text"`
	Chunks     string `yaml:"chunks"`
	Embeddings string `yaml:"embeddings"`
	Index      string `yaml:"index"`
}

type QdrantConfig struct {
	Host               string `yaml:"host"`
	Port               int    `yaml:"port"`
	MetadataCollection string `yaml:"metadata_collection"`
}

type ServicesConfig struct {
	STTEndpoint   string `yaml:"stt_endpoint"`
	EmbedEndpoint string `yaml:"embed_endpoint"`
}

type OrchestratorConfig struct {
	RetryLimit          int    `yaml:"retry_limit"`
	RetryDelaySeconds   int    `yaml:"retry_delay_seconds"`
	LogDir              string `yaml:"log_dir"`
}

type CorpusConfig struct {
	ID                   string                 `yaml:"id"`
	Type                 string                 `yaml:"type"`
	Mode                 string                 `yaml:"mode"`
	Schedule             string                 `yaml:"schedule,omitempty"`
	WatchIntervalSeconds int                    `yaml:"watch_interval_seconds,omitempty"`
	Config               map[string]interface{} `yaml:"config"`
}

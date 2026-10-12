package config

import "testing"

func TestExampleConfigContextSettings(t *testing.T) {
	cfg, err := Load("../../config/raggo.example.yml")
	if err != nil {
		t.Fatal(err)
	}
	if got := cfg.Storage.Documents.Contexts; got != "/var/lib/raggo/documents/contexts" {
		t.Errorf("documents contexts dir: %q", got)
	}
	if got := cfg.Storage.Podcast.Contexts; got != "/var/lib/raggo/podcast/contexts" {
		t.Errorf("podcast contexts dir: %q", got)
	}
	if ResolveContextEndpoint(cfg, "") == "" || ResolveContextModel(cfg, "") != "qwen3.5-4b" {
		t.Errorf("context service not loaded: %+v", cfg.Services)
	}
	if got := ResolveContextBudgetChars(cfg, 0); got != 12000 {
		t.Errorf("budget: %d", got)
	}
	if got := ResolveContextBudgetChars(cfg, 40000); got != 40000 {
		t.Errorf("flag override ignored: %d", got)
	}
}

func TestResolveTikaEndpoint(t *testing.T) {
	cfg, err := Load("../../config/raggo.example.yml")
	if err != nil {
		t.Fatal(err)
	}
	if got := ResolveTikaEndpoint(cfg, ""); got != "http://tika:9998" {
		t.Errorf("config endpoint not used: %q", got)
	}
	if got := ResolveTikaEndpoint(cfg, "http://other:9998"); got != "http://other:9998" {
		t.Errorf("flag override ignored: %q", got)
	}
	if got := ResolveTikaEndpoint(&Config{}, ""); got != "http://localhost:9998" {
		t.Errorf("default: %q", got)
	}
}

func TestResolveAPIKeyConfigWinsOverEnv(t *testing.T) {
	t.Setenv("RAGGO_EMBED_API_KEY", "from-env")
	t.Setenv("RAGGO_QDRANT_API_KEY", "qd-env")

	if got := ResolveEmbedAPIKey(nil); got != "from-env" {
		t.Errorf("nil config: %q", got)
	}
	cfg := &Config{}
	if got := ResolveEmbedAPIKey(cfg); got != "from-env" {
		t.Errorf("empty config value should fall back to env: %q", got)
	}
	cfg.Services.EmbedAPIKey = "from-config"
	if got := ResolveEmbedAPIKey(cfg); got != "from-config" {
		t.Errorf("config value should win: %q", got)
	}
	if got := ResolveQdrantAPIKey(cfg); got != "qd-env" {
		t.Errorf("qdrant: %q", got)
	}
	if got := ResolveSTTAPIKey(cfg); got != "" {
		t.Errorf("unset key should be empty: %q", got)
	}
}

func TestResolveEmbedAndSTTFormat(t *testing.T) {
	if got := ResolveEmbedAPI(nil, ""); got != "openai" {
		t.Errorf("embed api default: %q", got)
	}
	if got := ResolveSTTModel(&Config{}, ""); got != "whisper-1" {
		t.Errorf("stt model default: %q", got)
	}
	cfg := &Config{}
	cfg.Services.EmbedAPI = "raggo"
	cfg.Services.EmbedModel = "embed-qwen3-0.6b"
	cfg.Services.STTAPI = "openai"
	if got := ResolveEmbedAPI(cfg, ""); got != "raggo" {
		t.Errorf("embed api from config: %q", got)
	}
	if got := ResolveEmbedModel(cfg, ""); got != "embed-qwen3-0.6b" {
		t.Errorf("embed model from config: %q", got)
	}
	if got := ResolveEmbedModel(cfg, "flag-model"); got != "flag-model" {
		t.Errorf("flag override ignored: %q", got)
	}
	if got := ResolveSTTAPI(cfg, "raggo"); got != "raggo" {
		t.Errorf("flag override ignored: %q", got)
	}
}

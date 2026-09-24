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

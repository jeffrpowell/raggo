package llm

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCompleteSendsCacheAndThinkingFlags(t *testing.T) {
	var got ChatRequest
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
			t.Fatal(err)
		}
		w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"hello"}}]}`))
	}))
	defer srv.Close()

	out, err := NewClient(srv.URL, "qwen3.5-4b", 150, true).Complete(context.Background(), "sys", "user")
	if err != nil {
		t.Fatal(err)
	}
	if out != "hello" {
		t.Errorf("got %q", out)
	}
	if !got.CachePrompt || got.Temperature != 0 || got.MaxTokens != 150 || got.Model != "qwen3.5-4b" {
		t.Errorf("unexpected request: %+v", got)
	}
	if got.ChatTemplateKwargs["enable_thinking"] != false {
		t.Errorf("enable_thinking not disabled: %+v", got.ChatTemplateKwargs)
	}
	if len(got.Messages) != 2 || got.Messages[0].Role != "system" || got.Messages[1].Content != "user" {
		t.Errorf("unexpected messages: %+v", got.Messages)
	}
}

func TestCompleteReturnsHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "model loading", http.StatusServiceUnavailable)
	}))
	defer srv.Close()

	if _, err := NewClient(srv.URL, "m", 10, false).Complete(context.Background(), "s", "u"); err == nil {
		t.Fatal("expected error")
	}
}

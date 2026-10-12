package embed

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestEmbedFormats(t *testing.T) {
	tests := []struct {
		api      string
		response string
		wantKey  string
	}{
		{APIRaggo, `{"embedding":[0.5,0.25]}`, "text"},
		{APIOpenAI, `{"object":"list","data":[{"index":0,"embedding":[0.5,0.25]}]}`, "input"},
	}
	for _, tt := range tests {
		t.Run(tt.api, func(t *testing.T) {
			var got map[string]string
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if auth := r.Header.Get("Authorization"); auth != "Bearer k" {
					t.Errorf("auth header: %q", auth)
				}
				json.NewDecoder(r.Body).Decode(&got)
				w.Write([]byte(tt.response))
			}))
			defer srv.Close()

			c, err := NewClient(srv.URL, "k", tt.api, "m")
			if err != nil {
				t.Fatal(err)
			}
			vec, err := c.Embed("hello")
			if err != nil {
				t.Fatal(err)
			}
			if len(vec) != 2 || vec[0] != 0.5 || vec[1] != 0.25 {
				t.Errorf("vector: %v", vec)
			}
			if got[tt.wantKey] != "hello" || got["model"] != "m" {
				t.Errorf("request body: %v", got)
			}
		})
	}
}

func TestEmbedEmptyResponseIsError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"embedding":[0.1]}`)) // raggo shape sent to an openai client
	}))
	defer srv.Close()

	c, _ := NewClient(srv.URL, "", APIOpenAI, "m")
	if _, err := c.Embed("x"); err == nil {
		t.Fatal("expected error for a response with no data")
	}
}

func TestNewClientRejectsUnknownAPI(t *testing.T) {
	if _, err := NewClient("http://x", "", "ollama", "m"); err == nil {
		t.Fatal("expected error")
	}
}

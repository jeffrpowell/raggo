package tika

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestExtractTextEndpointSelection(t *testing.T) {
	tika4 := func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/tika/text":
			w.Header().Set("Content-Type", "text/plain;charset=UTF-8")
			w.Write([]byte("plain"))
		case "/tika":
			w.Write([]byte("# markdown"))
		}
	}
	tika3 := func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/tika/text":
			http.Error(w, "not acceptable", http.StatusNotAcceptable)
		case "/tika":
			w.Header().Set("Content-Type", "text/plain")
			w.Write([]byte("plain"))
		}
	}
	tika3JSON := func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/tika/text":
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"X-TIKA:content":"plain"}`))
		case "/tika":
			w.Header().Set("Content-Type", "text/plain")
			w.Write([]byte("plain"))
		}
	}

	cases := map[string]http.HandlerFunc{"tika 4": tika4, "tika 3": tika3, "tika 3 json": tika3JSON}
	for name, handler := range cases {
		t.Run(name, func(t *testing.T) {
			result, err := newTestClient(t, handler).ExtractText(writeTestFile(t))
			if err != nil {
				t.Fatal(err)
			}
			if result.Text != "plain" {
				t.Errorf("got %q, want %q", result.Text, "plain")
			}
			if result.Metadata["xmpTPg:NPages"] != "2" {
				t.Errorf("metadata not loaded: %v", result.Metadata)
			}
		})
	}
}

func TestExtractTextDoesNotFallBackOnServerError(t *testing.T) {
	calls := 0
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		calls++
		http.Error(w, "parse failed", http.StatusInternalServerError)
	})

	if _, err := client.ExtractText(writeTestFile(t)); err == nil {
		t.Fatal("expected error")
	}
	if calls != 1 {
		t.Errorf("got %d requests, want 1", calls)
	}
}

func newTestClient(t *testing.T, handler http.HandlerFunc) *Client {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/meta" {
			w.Write([]byte(`{"xmpTPg:NPages":"2"}`))
			return
		}
		handler(w, r)
	}))
	t.Cleanup(srv.Close)
	return NewClient(srv.URL)
}

func writeTestFile(t *testing.T) string {
	path := filepath.Join(t.TempDir(), "doc.pdf")
	if err := os.WriteFile(path, []byte("%PDF-1.4"), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

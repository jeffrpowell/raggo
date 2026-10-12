package stt

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func writeAudio(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "ep.mp3")
	if err := os.WriteFile(path, []byte("fake-audio"), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestTranscribeOpenAIUploadsFile(t *testing.T) {
	audio := writeAudio(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if auth := r.Header.Get("Authorization"); auth != "Bearer k" {
			t.Errorf("auth header: %q", auth)
		}
		if err := r.ParseMultipartForm(1 << 20); err != nil {
			t.Error(err)
			return
		}
		if got := r.FormValue("model"); got != "whisper" {
			t.Errorf("model: %q", got)
		}
		if got := r.FormValue("response_format"); got != "verbose_json" {
			t.Errorf("response_format: %q", got)
		}
		f, hdr, err := r.FormFile("file")
		if err != nil {
			t.Error(err)
			return
		}
		body, _ := io.ReadAll(f)
		if hdr.Filename != "ep.mp3" || string(body) != "fake-audio" {
			t.Errorf("file: %q %q", hdr.Filename, body)
		}
		w.Write([]byte(`{"text":" Hi there.","segments":[{"id":0,"start":0.0,"end":1.5,"text":" Hi"},{"id":1,"start":1.5,"end":2.0,"text":" there."}]}`))
	}))
	defer srv.Close()

	c, err := NewClient(srv.URL, "k", APIOpenAI, "whisper")
	if err != nil {
		t.Fatal(err)
	}
	segs, err := c.Transcribe(audio)
	if err != nil {
		t.Fatal(err)
	}
	if len(segs) != 2 || segs[1].Start != 1.5 || segs[1].End != 2.0 || segs[1].Text != " there." {
		t.Errorf("segments: %+v", segs)
	}
}

func TestTranscribeOpenAIRequiresSegments(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.Copy(io.Discard, r.Body)
		w.Write([]byte(`{"text":"plain json only"}`))
	}))
	defer srv.Close()

	c, _ := NewClient(srv.URL, "", APIOpenAI, "whisper")
	if _, err := c.Transcribe(writeAudio(t)); err == nil {
		t.Fatal("expected error when the server ignores verbose_json")
	}
}

func TestTranscribeOpenAIServerError(t *testing.T) {
	// The server answers before reading the upload; the writer goroutine
	// must not hang.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "nope", http.StatusBadRequest)
	}))
	defer srv.Close()

	c, _ := NewClient(srv.URL, "", APIOpenAI, "whisper")
	if _, err := c.Transcribe(writeAudio(t)); err == nil {
		t.Fatal("expected error")
	}
}

func TestTranscribeRaggoSendsPath(t *testing.T) {
	var got map[string]string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&got)
		w.Write([]byte(`{"segments":[{"start":0,"end":1,"text":"hi"}]}`))
	}))
	defer srv.Close()

	c, _ := NewClient(srv.URL, "", APIRaggo, "whisper")
	segs, err := c.Transcribe("/data/ep.mp3")
	if err != nil {
		t.Fatal(err)
	}
	if got["audio_path"] != "/data/ep.mp3" || len(segs) != 1 || segs[0].Text != "hi" {
		t.Errorf("request %v, segments %+v", got, segs)
	}
}

func TestNewClientRejectsUnknownAPI(t *testing.T) {
	if _, err := NewClient("http://x", "", "deepgram", "m"); err == nil {
		t.Fatal("expected error")
	}
}

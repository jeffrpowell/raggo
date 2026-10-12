package stt

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/jeffrpowell/raggo/pkg/schema"
)

// Request/response formats an STT endpoint can speak.
const (
	// APIRaggo POSTs {"audio_path"} (the service must see the same
	// filesystem) and expects {"segments":[...]}.
	APIRaggo = "raggo"
	// APIOpenAI uploads the audio as multipart form data to
	// /v1/audio/transcriptions (whisper.cpp server, llama-swap, OpenAI) with
	// response_format=verbose_json, which carries segment timestamps.
	APIOpenAI = "openai"
)

type Client struct {
	endpoint   string
	apiKey     string
	api        string
	model      string
	httpClient *http.Client
}

func NewClient(endpoint, apiKey, api, model string) (*Client, error) {
	if api != APIRaggo && api != APIOpenAI {
		return nil, fmt.Errorf("unknown stt api %q (want %q or %q)", api, APIRaggo, APIOpenAI)
	}
	// No timeout: transcribing a long episode can take a very long time.
	return &Client{endpoint: endpoint, apiKey: apiKey, api: api, model: model, httpClient: &http.Client{}}, nil
}

type raggoRequest struct {
	AudioPath string `json:"audio_path"`
}

type raggoResponse struct {
	Segments []schema.TranscriptSegment `json:"segments"`
}

type openAIResponse struct {
	Text     string `json:"text"`
	Segments []struct {
		Start float64 `json:"start"`
		End   float64 `json:"end"`
		Text  string  `json:"text"`
	} `json:"segments"`
}

func (c *Client) Transcribe(audioPath string) ([]schema.TranscriptSegment, error) {
	if c.api == APIOpenAI {
		return c.transcribeOpenAI(audioPath)
	}

	data, err := json.Marshal(raggoRequest{AudioPath: audioPath})
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest("POST", c.endpoint, bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	var resp raggoResponse
	if err := c.do(req, &resp); err != nil {
		return nil, err
	}
	return resp.Segments, nil
}

func (c *Client) transcribeOpenAI(audioPath string) ([]schema.TranscriptSegment, error) {
	f, err := os.Open(audioPath)
	if err != nil {
		return nil, err
	}

	// Stream the upload so a large episode is never held in memory. The
	// transport closes pr on any error, which unblocks the writer goroutine.
	pr, pw := io.Pipe()
	mw := multipart.NewWriter(pw)
	req, err := http.NewRequest("POST", c.endpoint, pr)
	if err != nil {
		f.Close()
		return nil, err
	}
	req.Header.Set("Content-Type", mw.FormDataContentType())

	go func() {
		defer f.Close()
		pw.CloseWithError(c.writeForm(mw, f, filepath.Base(audioPath)))
	}()

	var resp openAIResponse
	if err := c.do(req, &resp); err != nil {
		return nil, err
	}
	if len(resp.Segments) == 0 && strings.TrimSpace(resp.Text) != "" {
		return nil, fmt.Errorf("response has text but no segments; the server must support response_format=verbose_json")
	}

	segments := make([]schema.TranscriptSegment, len(resp.Segments))
	for i, s := range resp.Segments {
		segments[i] = schema.TranscriptSegment{Start: s.Start, End: s.End, Text: s.Text}
	}
	return segments, nil
}

func (c *Client) writeForm(mw *multipart.Writer, audio io.Reader, filename string) error {
	fields := [][2]string{
		{"model", c.model},
		{"response_format", "verbose_json"},
		{"timestamp_granularities[]", "segment"},
	}
	for _, kv := range fields {
		if err := mw.WriteField(kv[0], kv[1]); err != nil {
			return err
		}
	}
	part, err := mw.CreateFormFile("file", filename)
	if err != nil {
		return err
	}
	if _, err := io.Copy(part, audio); err != nil {
		return err
	}
	return mw.Close()
}

func (c *Client) do(req *http.Request, out interface{}) error {
	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("http %d: %s", resp.StatusCode, string(body))
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

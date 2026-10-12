package embed

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Request/response formats an embedding endpoint can speak.
const (
	// APIRaggo POSTs {"text","model"} and expects {"embedding":[...]}.
	APIRaggo = "raggo"
	// APIOpenAI POSTs {"input","model"} to /v1/embeddings (llama-server,
	// llama-swap, OpenAI) and expects {"data":[{"embedding":[...]}]}.
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
		return nil, fmt.Errorf("unknown embed api %q (want %q or %q)", api, APIRaggo, APIOpenAI)
	}
	return &Client{
		endpoint: endpoint,
		apiKey:   apiKey,
		api:      api,
		model:    model,
		httpClient: &http.Client{
			Timeout: 120 * time.Second,
		},
	}, nil
}

type raggoRequest struct {
	Text  string `json:"text"`
	Model string `json:"model"`
}

type raggoResponse struct {
	Embedding []float32 `json:"embedding"`
}

type openAIRequest struct {
	Input string `json:"input"`
	Model string `json:"model"`
}

type openAIResponse struct {
	Data []struct {
		Embedding []float32 `json:"embedding"`
	} `json:"data"`
}

func (c *Client) Embed(text string) ([]float32, error) {
	var vector []float32
	if c.api == APIOpenAI {
		var resp openAIResponse
		if err := c.post(openAIRequest{Input: text, Model: c.model}, &resp); err != nil {
			return nil, err
		}
		if len(resp.Data) > 0 {
			vector = resp.Data[0].Embedding
		}
	} else {
		var resp raggoResponse
		if err := c.post(raggoRequest{Text: text, Model: c.model}, &resp); err != nil {
			return nil, err
		}
		vector = resp.Embedding
	}
	if len(vector) == 0 {
		return nil, fmt.Errorf("empty embedding in %s response", c.api)
	}
	return vector, nil
}

func (c *Client) post(body, out interface{}) error {
	data, err := json.Marshal(body)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", c.endpoint, bytes.NewReader(data))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("http %d: %s", resp.StatusCode, string(respBody))
	}

	return json.NewDecoder(resp.Body).Decode(out)
}

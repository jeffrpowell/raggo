package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Client is a minimal OpenAI-compatible chat completions client tuned for
// llama.cpp's llama-server (prompt caching, chat template kwargs).
type Client struct {
	endpoint        string
	model           string
	maxTokens       int
	disableThinking bool
	httpClient      *http.Client
}

type ChatRequest struct {
	Model              string                 `json:"model"`
	Messages           []ChatMessage          `json:"messages"`
	MaxTokens          int                    `json:"max_tokens"`
	Temperature        float64                `json:"temperature"`
	CachePrompt        bool                   `json:"cache_prompt"`
	ChatTemplateKwargs map[string]interface{} `json:"chat_template_kwargs,omitempty"`
}

type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ChatResponse struct {
	Choices []ChatChoice `json:"choices"`
	Error   *ChatError   `json:"error,omitempty"`
}

type ChatChoice struct {
	Message ChatMessage `json:"message"`
}

type ChatError struct {
	Message string `json:"message"`
	Type    string `json:"type"`
}

func NewClient(endpoint, model string, maxTokens int, disableThinking bool) *Client {
	return &Client{
		endpoint:        endpoint,
		model:           model,
		maxTokens:       maxTokens,
		disableThinking: disableThinking,
		httpClient: &http.Client{
			Timeout: 600 * time.Second,
		},
	}
}

func (c *Client) Model() string {
	return c.model
}

// Complete sends a system + user message pair and returns the assistant text.
// Callers should put stable content first in the user message so llama-server
// can reuse its KV cache across consecutive calls.
func (c *Client) Complete(ctx context.Context, system, user string) (string, error) {
	req := ChatRequest{
		Model: c.model,
		Messages: []ChatMessage{
			{Role: "system", Content: system},
			{Role: "user", Content: user},
		},
		MaxTokens:   c.maxTokens,
		Temperature: 0,
		CachePrompt: true,
	}
	if c.disableThinking {
		req.ChatTemplateKwargs = map[string]interface{}{"enable_thinking": false}
	}

	reqBody, err := json.Marshal(req)
	if err != nil {
		return "", fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.endpoint, bytes.NewReader(reqBody))
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("llm request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("llm returned status %d: %s", resp.StatusCode, string(body))
	}

	var apiResp ChatResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return "", fmt.Errorf("decode response: %w", err)
	}

	if apiResp.Error != nil {
		return "", fmt.Errorf("llm error: %s", apiResp.Error.Message)
	}

	if len(apiResp.Choices) == 0 {
		return "", fmt.Errorf("no choices in response")
	}

	return apiResp.Choices[0].Message.Content, nil
}

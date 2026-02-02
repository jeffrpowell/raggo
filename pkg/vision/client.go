package vision

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"image"
	"image/png"
	"io"
	"net/http"
	"time"

	"github.com/gen2brain/go-fitz"
)

type Client struct {
	endpoint   string
	model      string
	prompt     string
	httpClient *http.Client
}

type OpenAIRequest struct {
	Model      string          `json:"model"`
	Messages   []OpenAIMessage `json:"messages"`
	MaxTokens  int             `json:"max_tokens"`
}

type OpenAIMessage struct {
	Role    string                `json:"role"`
	Content []OpenAIContentPart   `json:"content"`
}

type OpenAIContentPart struct {
	Type     string            `json:"type"`
	Text     string            `json:"text,omitempty"`
	ImageURL *OpenAIImageURL   `json:"image_url,omitempty"`
}

type OpenAIImageURL struct {
	URL string `json:"url"`
}

type OpenAIResponse struct {
	Choices []OpenAIChoice `json:"choices"`
	Error   *OpenAIError   `json:"error,omitempty"`
}

type OpenAIChoice struct {
	Message OpenAIResponseMessage `json:"message"`
}

type OpenAIResponseMessage struct {
	Content string `json:"content"`
}

type OpenAIError struct {
	Message string `json:"message"`
	Type    string `json:"type"`
}

func NewClient(endpoint, model, prompt string) *Client {
	return &Client{
		endpoint: endpoint,
		model:    model,
		prompt:   prompt,
		httpClient: &http.Client{
			Timeout: 300 * time.Second,
		},
	}
}

func (c *Client) ExtractTextFromPDF(filePath string) (string, error) {
	doc, err := fitz.New(filePath)
	if err != nil {
		return "", fmt.Errorf("open pdf: %w", err)
	}
	defer doc.Close()

	var allText bytes.Buffer
	numPages := doc.NumPage()

	for pageNum := 0; pageNum < numPages; pageNum++ {
		img, err := doc.Image(pageNum)
		if err != nil {
			return "", fmt.Errorf("render page %d: %w", pageNum, err)
		}

		pageText, err := c.ExtractTextFromImage(img)
		if err != nil {
			return "", fmt.Errorf("ocr page %d: %w", pageNum, err)
		}

		allText.WriteString(pageText)
		allText.WriteString("\n\n")
	}

	return allText.String(), nil
}

func (c *Client) ExtractTextFromImage(img image.Image) (string, error) {
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return "", fmt.Errorf("encode png: %w", err)
	}

	base64Img := base64.StdEncoding.EncodeToString(buf.Bytes())
	dataURL := fmt.Sprintf("data:image/png;base64,%s", base64Img)

	req := OpenAIRequest{
		Model: c.model,
		Messages: []OpenAIMessage{
			{
				Role: "user",
				Content: []OpenAIContentPart{
					{
						Type: "text",
						Text: c.prompt,
					},
					{
						Type: "image_url",
						ImageURL: &OpenAIImageURL{
							URL: dataURL,
						},
					},
				},
			},
		},
		MaxTokens: 4096,
	}

	reqBody, err := json.Marshal(req)
	if err != nil {
		return "", fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequest("POST", c.endpoint, bytes.NewReader(reqBody))
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("vision api request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("vision api returned status %d: %s", resp.StatusCode, string(body))
	}

	var apiResp OpenAIResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return "", fmt.Errorf("decode response: %w", err)
	}

	if apiResp.Error != nil {
		return "", fmt.Errorf("vision api error: %s", apiResp.Error.Message)
	}

	if len(apiResp.Choices) == 0 {
		return "", fmt.Errorf("no choices in response")
	}

	return apiResp.Choices[0].Message.Content, nil
}

func (c *Client) Ping() error {
	req := OpenAIRequest{
		Model: c.model,
		Messages: []OpenAIMessage{
			{
				Role: "user",
				Content: []OpenAIContentPart{
					{
						Type: "text",
						Text: "test",
					},
				},
			},
		},
		MaxTokens: 1,
	}

	reqBody, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequest("POST", c.endpoint, bytes.NewReader(reqBody))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("ping failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusBadRequest {
		return fmt.Errorf("vision api not ready, status: %d", resp.StatusCode)
	}

	return nil
}

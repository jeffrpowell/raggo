package tika

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

type Client struct {
	endpoint   string
	httpClient *http.Client
}

type ExtractionResult struct {
	Text     string
	Metadata map[string]interface{}
}

func NewClient(endpoint string) *Client {
	return &Client{
		endpoint: endpoint,
		httpClient: &http.Client{
			Timeout: 120 * time.Second,
		},
	}
}

func (c *Client) ExtractText(filePath string) (*ExtractionResult, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("open file: %w", err)
	}
	defer file.Close()

	url := fmt.Sprintf("%s/tika", c.endpoint)
	req, err := http.NewRequest("PUT", url, file)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Accept", "text/plain")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("tika request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("tika returned status %d: %s", resp.StatusCode, string(body))
	}

	text, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	metadata, err := c.extractMetadata(filePath)
	if err != nil {
		return nil, fmt.Errorf("extract metadata: %w", err)
	}

	return &ExtractionResult{
		Text:     string(text),
		Metadata: metadata,
	}, nil
}

func (c *Client) extractMetadata(filePath string) (map[string]interface{}, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("open file: %w", err)
	}
	defer file.Close()

	url := fmt.Sprintf("%s/meta", c.endpoint)
	req, err := http.NewRequest("PUT", url, file)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("metadata request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("tika metadata returned status %d: %s", resp.StatusCode, string(body))
	}

	var metadata map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&metadata); err != nil {
		return nil, fmt.Errorf("decode metadata: %w", err)
	}

	return metadata, nil
}

func (c *Client) Ping() error {
	url := fmt.Sprintf("%s/tika", c.endpoint)
	resp, err := c.httpClient.Get(url)
	if err != nil {
		return fmt.Errorf("ping failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("tika not ready, status: %d", resp.StatusCode)
	}

	return nil
}

func ExtractPageCount(metadata map[string]interface{}) int {
	if val, ok := metadata["xmpTPg:NPages"]; ok {
		if count, err := interfaceToInt(val); err == nil {
			return count
		}
	}
	if val, ok := metadata["meta:page-count"]; ok {
		if count, err := interfaceToInt(val); err == nil {
			return count
		}
	}
	if val, ok := metadata["Page-Count"]; ok {
		if count, err := interfaceToInt(val); err == nil {
			return count
		}
	}
	return 0
}

func ExtractImageCount(metadata map[string]interface{}) int {
	if val, ok := metadata["meta:image-count"]; ok {
		if count, err := interfaceToInt(val); err == nil {
			return count
		}
	}
	return 0
}

func HasImages(metadata map[string]interface{}) bool {
	return ExtractImageCount(metadata) > 0
}

func interfaceToInt(val interface{}) (int, error) {
	switch v := val.(type) {
	case int:
		return v, nil
	case int64:
		return int(v), nil
	case float64:
		return int(v), nil
	case string:
		return strconv.Atoi(strings.TrimSpace(v))
	case []interface{}:
		if len(v) > 0 {
			return interfaceToInt(v[0])
		}
	}
	return 0, fmt.Errorf("cannot convert %T to int", val)
}

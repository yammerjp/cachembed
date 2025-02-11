package upstream

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

// Client はOpenAI APIクライアントの構造体
type Client struct {
	httpClient *http.Client
	baseURL    string
}

// NewClient は新しいClientを作成します
func NewClient(httpClient *http.Client, baseURL string) *Client {
	return &Client{
		httpClient: httpClient,
		baseURL:    baseURL,
	}
}

// CreateEmbedding は埋め込みを作成します
func (c *Client) CreateEmbedding(req *EmbeddingRequest, authHeader string) (*EmbeddingResponse, error) {
	jsonData, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequest("POST", c.baseURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", authHeader)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errResp ErrorResponse
		if err := json.NewDecoder(resp.Body).Decode(&errResp); err != nil {
			return nil, fmt.Errorf("failed to decode error response: %w", err)
		}
		return nil, &UpstreamError{
			StatusCode: resp.StatusCode,
			ErrorInfo: struct {
				Message string `json:"message"`
				Type    string `json:"type"`
			}{
				Message: errResp.Error.Message,
				Type:    errResp.Error.Type,
			},
			Usage: errResp.Usage,
		}
	}

	var embedResp EmbeddingResponse
	if err := json.NewDecoder(resp.Body).Decode(&embedResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &embedResp, nil
}

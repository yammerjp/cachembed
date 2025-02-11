package upstream

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

// ErrorResponse はOpenAI APIからのエラーレスポンスの構造体
type ErrorResponse struct {
	Error struct {
		Message string `json:"message"`
		Type    string `json:"type"`
		Code    string `json:"code,omitempty"`
	} `json:"error"`
}

// Client はOpenAI APIクライアントの構造体
type Client struct {
	baseURL string
}

// NewClient は新しいClientを作成します
func NewClient(baseURL string) *Client {
	return &Client{
		baseURL: baseURL,
	}
}

// CreateEmbedding は埋め込みを作成します
func (c *Client) CreateEmbedding(req *EmbeddingRequest, authHeader string) (*EmbeddingResponse, error) {
	jsonData, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequest("POST", c.baseURL+"/v1/embeddings", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", authHeader)

	client := &http.Client{}
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errResp ErrorResponse
		if err := json.NewDecoder(resp.Body).Decode(&errResp); err != nil {
			return nil, &UpstreamError{
				StatusCode: resp.StatusCode,
				Response: map[string]interface{}{
					"error": map[string]interface{}{
						"message": "Failed to decode error response",
						"type":    "internal_error",
					},
				},
			}
		}
		return nil, &UpstreamError{
			StatusCode: resp.StatusCode,
			Response: map[string]interface{}{
				"error": map[string]interface{}{
					"message": errResp.Error.Message,
					"type":    errResp.Error.Type,
					"code":    errResp.Error.Code,
				},
			},
		}
	}

	var embedResp EmbeddingResponse
	if err := json.NewDecoder(resp.Body).Decode(&embedResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &embedResp, nil
}

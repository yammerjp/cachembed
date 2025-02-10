package upstream

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"time"
)

type EmbeddingRequest struct {
	Input          interface{} `json:"input"`
	Model          string      `json:"model"`
	EncodingFormat string      `json:"encoding_format,omitempty"`
	Dimensions     *int        `json:"dimensions,omitempty"`
	User           string      `json:"user,omitempty"`
}

type ErrorResponse struct {
	Error struct {
		Message string `json:"message"`
		Type    string `json:"type"`
		Code    string `json:"code"`
	} `json:"error"`
}

type EmbeddingResponse struct {
	Object string `json:"object"`
	Data   []struct {
		Object    string      `json:"object"`
		Embedding interface{} `json:"embedding"`
		Index     int         `json:"index"`
	} `json:"data"`
	Model string `json:"model"`
	Usage struct {
		PromptTokens int `json:"prompt_tokens"`
		TotalTokens  int `json:"total_tokens"`
	} `json:"usage"`
}

type Client struct {
	url        string
	httpClient *http.Client
}

func NewClient(url string) *Client {
	return &Client{
		url: url,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (c *Client) CreateEmbedding(req *EmbeddingRequest, authHeader string) (*EmbeddingResponse, error) {
	requestBody, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	upstreamReq, err := http.NewRequest(http.MethodPost, c.url, bytes.NewBuffer(requestBody))
	if err != nil {
		return nil, err
	}

	upstreamReq.Header.Set("Content-Type", "application/json")
	upstreamReq.Header.Set("Authorization", authHeader)

	resp, err := c.httpClient.Do(upstreamReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		// エラーレスポンスをそのまま返す
		var errResp ErrorResponse
		if err := json.Unmarshal(respBody, &errResp); err != nil {
			return nil, err
		}
		return nil, &UpstreamError{
			StatusCode: resp.StatusCode,
			Response:   &errResp,
		}
	}

	var embeddingResp EmbeddingResponse
	if err := json.Unmarshal(respBody, &embeddingResp); err != nil {
		return nil, err
	}

	return &embeddingResp, nil
}

type UpstreamError struct {
	StatusCode int
	Response   *ErrorResponse
}

func (e *UpstreamError) Error() string {
	return e.Response.Error.Message
}

package main

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"time"
)

type upstreamClient struct {
	url        string
	httpClient *http.Client
}

func newUpstreamClient(url string) *upstreamClient {
	return &upstreamClient{
		url: url,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (c *upstreamClient) createEmbedding(req *EmbeddingRequest, authHeader string) (*EmbeddingResponse, error) {
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
		return nil, &upstreamError{
			statusCode: resp.StatusCode,
			response:   &errResp,
		}
	}

	var embeddingResp EmbeddingResponse
	if err := json.Unmarshal(respBody, &embeddingResp); err != nil {
		return nil, err
	}

	return &embeddingResp, nil
}

type upstreamError struct {
	statusCode int
	response   *ErrorResponse
}

func (e *upstreamError) Error() string {
	return e.response.Error.Message
}

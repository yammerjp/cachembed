package upstream

import (
	"fmt"
)

// EmbeddingRequest は埋め込みリクエストの構造体
type EmbeddingRequest struct {
	Input          interface{} `json:"input"`
	Model          string      `json:"model"`
	EncodingFormat string      `json:"encoding_format,omitempty"`
}

// EmbeddingResponse は埋め込みレスポンスの構造体
type EmbeddingResponse struct {
	Object string          `json:"object"`
	Data   []EmbeddingData `json:"data"`
	Model  string          `json:"model"`
	Usage  Usage           `json:"usage"`
}

// EmbeddingData は個々の埋め込みデータの構造体
type EmbeddingData struct {
	Object    string      `json:"object"`
	Embedding interface{} `json:"embedding"`
	Index     int         `json:"index"`
}

// Usage はトークン使用量の構造体
type Usage struct {
	PromptTokens int `json:"prompt_tokens"`
	TotalTokens  int `json:"total_tokens"`
}

// UpstreamError はアップストリームAPIからのエラーレスポンスの構造体
type UpstreamError struct {
	StatusCode int
	Response   map[string]interface{}
}

func (e *UpstreamError) Error() string {
	return fmt.Sprintf("upstream error: status code %d", e.StatusCode)
}

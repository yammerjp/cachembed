package upstream

import (
	"fmt"
)

// EmbeddingRequest は埋め込みベクトル生成リクエストを表します
type EmbeddingRequest struct {
	Input          interface{} `json:"input"`
	Model          string      `json:"model"`
	EncodingFormat string      `json:"encoding_format,omitempty"`
	Dimension      int         `json:"dimension,omitempty"`
}

// EmbeddingResponse は埋め込みベクトル生成レスポンスを表します
type EmbeddingResponse struct {
	Object string          `json:"object"`
	Data   []EmbeddingData `json:"data"`
	Model  string          `json:"model"`
	Usage  Usage           `json:"usage"`
}

// EmbeddingData は個々の埋め込みベクトルデータを表します
type EmbeddingData struct {
	Object    string      `json:"object"`
	Embedding interface{} `json:"embedding"`
	Index     int         `json:"index"`
}

// Usage はトークン使用量を表します
type Usage struct {
	PromptTokens int `json:"prompt_tokens"`
	TotalTokens  int `json:"total_tokens"`
}

// ErrorResponse はOpenAI APIからのエラーレスポンスの構造体
type ErrorResponse struct {
	Error struct {
		Message string `json:"message"`
		Type    string `json:"type"`
		Code    string `json:"code,omitempty"`
	} `json:"error"`
	Usage Usage `json:"usage"`
}

// UpstreamError は上流サーバーからのエラーレスポンスを表します
type UpstreamError struct {
	StatusCode int `json:"status_code"`
	ErrorInfo  struct {
		Message string `json:"message"`
		Type    string `json:"type"`
	} `json:"error"`
	Usage Usage `json:"usage"`
}

func (e *UpstreamError) Error() string {
	return fmt.Sprintf("%s: %s", e.ErrorInfo.Type, e.ErrorInfo.Message)
}

// ErrNotFound はリソースが見つからない場合のエラーです
var ErrNotFound = fmt.Errorf("not found")

// EmbeddingClient は埋め込みベクトル生成クライアントのインターフェースです
type EmbeddingClient interface {
	CreateEmbedding(req *EmbeddingRequest, authHeader string) (*EmbeddingResponse, error)
}

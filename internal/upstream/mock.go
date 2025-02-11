package upstream

import (
	"fmt"
	"math/rand/v2"
)

// MockClient はテスト用のモッククライアント
type MockClient struct{}

func NewMockClient() *MockClient {
	return &MockClient{}
}

func (c *MockClient) CreateEmbedding(req *EmbeddingRequest, authHeader string) (*EmbeddingResponse, error) {
	// エラーケースの処理
	if req.Model == "error-model" {
		return nil, &UpstreamError{
			StatusCode: 400,
			ErrorInfo: struct {
				Message string `json:"message"`
				Type    string `json:"type"`
			}{
				Message: "invalid model",
				Type:    "invalid_request_error",
			},
		}
	}

	// 入力の検証
	var inputLength int
	switch v := req.Input.(type) {
	case string:
		inputLength = 1
	case []interface{}:
		inputLength = len(v)
		// float値のバリデーション
		for i, item := range v {
			if num, ok := item.(float64); ok {
				if float64(int(num)) != num {
					return nil, &UpstreamError{
						StatusCode: 400,
						ErrorInfo: struct {
							Message string `json:"message"`
							Type    string `json:"type"`
						}{
							Message: fmt.Sprintf("non-integer number at index %d: %v", i, num),
							Type:    "invalid_request_error",
						},
					}
				}
			}
		}
	default:
		return nil, fmt.Errorf("unsupported input type: %T", req.Input)
	}

	data := make([]EmbeddingData, inputLength)
	for i := range data {
		mockEmbedding := make([]float32, inputLength)
		for i := range mockEmbedding {
			mockEmbedding[i] = rand.Float32()
		}

		data[i] = EmbeddingData{
			Object: "embedding",
			Index:  i,
		}
		if req.EncodingFormat == "base64" {
			data[i].Embedding = float32ToBase64(mockEmbedding)
		} else {
			data[i].Embedding = mockEmbedding
		}
	}

	return &EmbeddingResponse{
		Object: "list",
		Data:   data,
		Model:  req.Model,
		Usage: Usage{
			PromptTokens: inputLength,
			TotalTokens:  inputLength,
		},
	}, nil
}

// MockClient が EmbeddingClient インターフェースを実装していることを確認
var _ EmbeddingClient = (*MockClient)(nil)

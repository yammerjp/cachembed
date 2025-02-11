package upstream

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCreateEmbedding(t *testing.T) {
	tests := []struct {
		name           string
		request        *EmbeddingRequest
		authHeader     string
		mockResponse   func(w http.ResponseWriter, r *http.Request)
		wantError      bool
		wantStatusCode int
		wantErrorType  string
	}{
		{
			name: "single string input success",
			request: &EmbeddingRequest{
				Input: "Hello, world",
				Model: "text-embedding-ada-002",
			},
			authHeader: "Bearer test-key",
			mockResponse: func(w http.ResponseWriter, r *http.Request) {
				// リクエストの検証
				if r.Header.Get("Authorization") != "Bearer test-key" {
					t.Error("Authorization header mismatch")
				}
				if r.Header.Get("Content-Type") != "application/json" {
					t.Error("Content-Type header mismatch")
				}

				// リクエストボディの検証
				body, _ := io.ReadAll(r.Body)
				var req EmbeddingRequest
				if err := json.Unmarshal(body, &req); err != nil {
					t.Errorf("Failed to parse request body: %v", err)
				}
				if req.Input != "Hello, world" {
					t.Errorf("Input mismatch: got %v", req.Input)
				}

				// 正常なレスポンスを返す
				json.NewEncoder(w).Encode(EmbeddingResponse{
					Object: "list",
					Data: []EmbeddingData{
						{
							Object: "embedding",
							Embedding: []float32{0.1,
								0.2,
								0.3},
							Index: 0,
						},
					},
					Model: "text-embedding-ada-002",
					Usage: Usage{
						PromptTokens: 8,
						TotalTokens:  8,
					},
				})
			},
			wantError: false,
		},
		{
			name: "multiple string input success",
			request: &EmbeddingRequest{
				Input: []string{"Hello", "World"},
				Model: "text-embedding-ada-002",
			},
			authHeader: "Bearer test-key",
			mockResponse: func(w http.ResponseWriter, r *http.Request) {
				json.NewEncoder(w).Encode(EmbeddingResponse{
					Object: "list",
					Data: []EmbeddingData{
						{
							Object: "embedding",
							Embedding: []float32{0.1,
								0.2},
							Index: 0,
						},
						{
							Object: "embedding",
							Embedding: []float32{0.3,
								0.4},
							Index: 1,
						},
					},
					Model: "text-embedding-ada-002",
					Usage: Usage{
						PromptTokens: 4,
						TotalTokens:  4,
					},
				})
			},
			wantError: false,
		},
		{
			name: "base64 encoding success",
			request: &EmbeddingRequest{
				Input:          "Hello, world",
				Model:          "text-embedding-ada-002",
				EncodingFormat: "base64",
			},
			authHeader: "Bearer test-key",
			mockResponse: func(w http.ResponseWriter, r *http.Request) {
				json.NewEncoder(w).Encode(EmbeddingResponse{
					Object: "list",
					Data: []EmbeddingData{
						{
							Object:    "embedding",
							Embedding: "base64encodedstring",
							Index:     0,
						},
					},
					Model: "text-embedding-ada-002",
					Usage: Usage{
						PromptTokens: 8,
						TotalTokens:  8,
					},
				})
			},
			wantError: false,
		},
		{
			name: "bad request error",
			request: &EmbeddingRequest{
				Input: "Hello, world",
				Model: "invalid-model",
			},
			authHeader: "Bearer test-key",
			mockResponse: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusBadRequest)
				json.NewEncoder(w).Encode(ErrorResponse{
					Error: struct {
						Message string `json:"message"`
						Type    string `json:"type"`
						Code    string `json:"code,omitempty"`
					}{
						Message: "Invalid model",
						Type:    "invalid_request_error",
					},
					Usage: Usage{},
				})
			},
			wantError:      true,
			wantStatusCode: http.StatusBadRequest,
			wantErrorType:  "invalid_request_error",
		},
		{
			name: "unauthorized error",
			request: &EmbeddingRequest{
				Input: "Hello, world",
				Model: "text-embedding-ada-002",
			},
			authHeader: "Bearer invalid-key",
			mockResponse: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusUnauthorized)
				json.NewEncoder(w).Encode(ErrorResponse{
					Error: struct {
						Message string `json:"message"`
						Type    string `json:"type"`
						Code    string `json:"code,omitempty"`
					}{
						Message: "Invalid authentication credentials",
						Type:    "invalid_request_error",
					},
					Usage: Usage{},
				})
			},
			wantError:      true,
			wantStatusCode: http.StatusUnauthorized,
			wantErrorType:  "invalid_request_error",
		},
		{
			name: "invalid json response",
			request: &EmbeddingRequest{
				Input: "Hello, world",
				Model: "text-embedding-ada-002",
			},
			authHeader: "Bearer test-key",
			mockResponse: func(w http.ResponseWriter, r *http.Request) {
				w.Write([]byte("invalid json"))
			},
			wantError: true,
		},
		{
			name: "network error",
			request: &EmbeddingRequest{
				Input: "Hello, world",
				Model: "text-embedding-ada-002",
			},
			authHeader: "Bearer test-key",
			mockResponse: func(w http.ResponseWriter, r *http.Request) {
				panic("simulated network error")
			},
			wantError: true,
		},
		{
			name: "string input",
			request: &EmbeddingRequest{
				Input: "こんにちは世界",
				Model: "text-embedding-ada-002",
			},
			authHeader: "Bearer test-key",
			mockResponse: func(w http.ResponseWriter, r *http.Request) {
				// リクエストボディの検証
				body, _ := io.ReadAll(r.Body)
				var req EmbeddingRequest
				if err := json.Unmarshal(body, &req); err != nil {
					t.Errorf("Failed to parse request body: %v", err)
				}
				if req.Input != "こんにちは世界" {
					t.Errorf("Input mismatch: got %v", req.Input)
				}

				json.NewEncoder(w).Encode(EmbeddingResponse{
					Object: "list",
					Data: []EmbeddingData{
						{
							Object: "embedding",
							Embedding: []float32{0.1,
								0.2,
								0.3},
							Index: 0,
						},
					},
					Model: "text-embedding-ada-002",
					Usage: Usage{
						PromptTokens: 8,
						TotalTokens:  8,
					},
				})
			},
			wantError: false,
		},
		{
			name: "integer array input (token sequence)",
			request: &EmbeddingRequest{
				Input: []int{1, 2, 3, 4, 5},
				Model: "text-embedding-ada-002",
			},
			authHeader: "Bearer test-key",
			mockResponse: func(w http.ResponseWriter, r *http.Request) {
				// リクエストボディの検証
				body, _ := io.ReadAll(r.Body)
				var req map[string]interface{}
				if err := json.Unmarshal(body, &req); err != nil {
					t.Errorf("Failed to parse request body: %v", err)
				}
				input, ok := req["input"].([]interface{})
				if !ok {
					t.Error("Expected input to be array")
					return
				}
				for i, v := range input {
					if int(v.(float64)) != i+1 {
						t.Errorf("Input mismatch at index %d: got %v, want %d", i, v, i+1)
					}
				}

				json.NewEncoder(w).Encode(EmbeddingResponse{
					Object: "list",
					Data: []EmbeddingData{
						{
							Object: "embedding",
							Embedding: []float32{0.1,
								0.2,
								0.3},
							Index: 0,
						},
					},
					Model: "text-embedding-ada-002",
					Usage: Usage{
						PromptTokens: 5,
						TotalTokens:  5,
					},
				})
			},
			wantError: false,
		},
		{
			name: "string array input",
			request: &EmbeddingRequest{
				Input: []string{"Hello", "World", "こんにちは"},
				Model: "text-embedding-ada-002",
			},
			authHeader: "Bearer test-key",
			mockResponse: func(w http.ResponseWriter, r *http.Request) {
				// リクエストボディの検証
				body, _ := io.ReadAll(r.Body)
				var req EmbeddingRequest
				if err := json.Unmarshal(body, &req); err != nil {
					t.Errorf("Failed to parse request body: %v", err)
				}
				input, ok := req.Input.([]interface{})
				if !ok {
					t.Error("Expected input to be array")
					return
				}
				expectedInputs := []string{"Hello", "World", "こんにちは"}
				for i, v := range input {
					if v != expectedInputs[i] {
						t.Errorf("Input mismatch at index %d: got %v, want %v", i, v, expectedInputs[i])
					}
				}

				json.NewEncoder(w).Encode(EmbeddingResponse{
					Object: "list",
					Data: []EmbeddingData{
						{
							Object: "embedding",
							Embedding: []float32{0.1,
								0.2,
								0.3},
							Index: 0,
						},
						{
							Object: "embedding",
							Embedding: []float32{0.4,
								0.5,
								0.6},
							Index: 1,
						},
						{
							Object: "embedding",
							Embedding: []float32{0.7,
								0.8,
								0.9},
							Index: 2,
						},
					},
					Model: "text-embedding-ada-002",
					Usage: Usage{
						PromptTokens: 6,
						TotalTokens:  6,
					},
				})
			},
			wantError: false,
		},
		{
			name: "array of integer arrays (token sequense array) input",
			request: &EmbeddingRequest{
				Input: [][]int{{1, 2}, {3, 4}, {5, 6}},
				Model: "text-embedding-ada-002",
			},
			authHeader: "Bearer test-key",
			mockResponse: func(w http.ResponseWriter, r *http.Request) {
				// リクエストボディの検証
				body, _ := io.ReadAll(r.Body)
				var req map[string]interface{}
				if err := json.Unmarshal(body, &req); err != nil {
					t.Errorf("Failed to parse request body: %v", err)
				}
				input, ok := req["input"].([]interface{})
				if !ok {
					t.Error("Expected input to be array")
					return
				}
				expectedInputs := [][]int{{1, 2}, {3, 4}, {5, 6}}
				for i, v := range input {
					arr, ok := v.([]interface{})
					if !ok {
						t.Errorf("Expected array at index %d", i)
						continue
					}
					for j, num := range arr {
						if int(num.(float64)) != expectedInputs[i][j] {
							t.Errorf("Input mismatch at [%d][%d]: got %v, want %d", i, j, num, expectedInputs[i][j])
						}
					}
				}

				json.NewEncoder(w).Encode(EmbeddingResponse{
					Object: "list",
					Data: []EmbeddingData{
						{
							Object: "embedding",
							Embedding: []float32{0.1,
								0.2},
							Index: 0,
						},
						{
							Object: "embedding",
							Embedding: []float32{0.3,
								0.4},
							Index: 1,
						},
						{
							Object: "embedding",
							Embedding: []float32{0.5,
								0.6},
							Index: 2,
						},
					},
					Model: "text-embedding-ada-002",
					Usage: Usage{
						PromptTokens: 6,
						TotalTokens:  6,
					},
				})
			},
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// モックサーバーの設定
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				defer func() {
					if r := recover(); r != nil && tt.name != "network error" {
						t.Errorf("Handler panic: %v", r)
					}
				}()
				tt.mockResponse(w, r)
			}))
			defer server.Close()

			// テスト用のクライアントを作成
			var client *Client
			if tt.name == "network error" {
				// ネットワークエラーをシミュレートするために無効なURLを使用
				client = NewClient(http.DefaultClient, "http://invalid-url")
			} else {
				client = NewClient(server.Client(), server.URL)
			}

			// テストの実行
			resp, err := client.CreateEmbedding(tt.request, tt.authHeader)

			// エラーの検証
			if (err != nil) != tt.wantError {
				t.Errorf("CreateEmbedding() error = %v, wantError %v", err, tt.wantError)
				return
			}

			if err != nil {
				if tt.wantStatusCode > 0 {
					if ue, ok := err.(*UpstreamError); ok {
						if ue.StatusCode != tt.wantStatusCode {
							t.Errorf("Wrong status code: got %v want %v", ue.StatusCode, tt.wantStatusCode)
						}
						if ue.ErrorInfo.Type != tt.wantErrorType {
							t.Errorf("Wrong error type: got %v want %v", ue.ErrorInfo.Type, tt.wantErrorType)
						}
					} else {
						t.Errorf("Expected UpstreamError but got %T", err)
					}
				}
				return
			}

			// 正常系の場合はレスポンスの検証
			if resp == nil {
				t.Error("Expected response but got nil")
				return
			}

			if resp.Model != tt.request.Model {
				t.Errorf("Wrong model in response: got %v want %v", resp.Model, tt.request.Model)
			}

			if len(resp.Data) == 0 {
				t.Error("Expected embedding data but got none")
			}
		})
	}
}

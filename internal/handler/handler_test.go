package handler

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/yammerjp/cachembed/internal/storage"
	"github.com/yammerjp/cachembed/internal/upstream"
)

func TestHandleEmbeddings(t *testing.T) {
	// テスト用の一時データベースを作成
	tmpFile, err := os.CreateTemp("", "cachembed-test-*.db")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	db, err := storage.NewDB(tmpFile.Name())
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	// モックサーバーの設定（シンプルな成功レスポンスのみ）
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// デバッグ用にリクエストの内容を表示
		slog.Debug("mock server received request",
			"method", r.Method,
			"path", r.URL.Path,
			"headers", r.Header,
		)

		// JSONとして正しい形式でレスポンスを構築
		rawResp := map[string]interface{}{
			"object": "list",
			"data": []map[string]interface{}{
				{
					"object":    "embedding",
					"embedding": []float64{0.1, 0.2, 0.3}, // float64を使用
					"index":     0,
				},
			},
			"model": "text-embedding-ada-002",
			"usage": map[string]interface{}{
				"prompt_tokens": 8,
				"total_tokens":  8,
			},
		}
		// デバッグ用にレスポンスの内容を表示
		respBytes, _ := json.Marshal(rawResp)
		slog.Debug("mock server sending response",
			"response", string(respBytes),
		)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(rawResp)
	}))
	defer ts.Close()

	allowedModels := []string{"text-embedding-ada-002"}
	apiKeyPattern := "^sk-[a-zA-Z0-9]{32}$"
	validAPIKey := "sk-abcdefghijklmnopqrstuvwxyz123456" // 有効なAPIキーの例

	tests := []struct {
		name          string
		request       map[string]interface{}
		authHeader    string
		wantStatus    int
		wantCacheHit  bool
		wantTokens    int
		wantErrorType string
	}{
		{
			name: "valid string input",
			request: map[string]interface{}{
				"input": "The food was delicious",
				"model": "text-embedding-ada-002",
			},
			authHeader:   "Bearer " + validAPIKey,
			wantStatus:   http.StatusOK,
			wantTokens:   8,
			wantCacheHit: false,
		},
		{
			name: "valid string input (cache hit)",
			request: map[string]interface{}{
				"input": "The food was delicious",
				"model": "text-embedding-ada-002",
			},
			authHeader:   "Bearer " + validAPIKey,
			wantStatus:   http.StatusOK,
			wantCacheHit: true,
			wantTokens:   0,
		},
		{
			name: "valid integer array input",
			request: map[string]interface{}{
				"input": []int{1, 2, 3, 4},
				"model": "text-embedding-ada-002",
			},
			authHeader:   "Bearer " + validAPIKey,
			wantStatus:   http.StatusOK,
			wantTokens:   8,
			wantCacheHit: false,
		},
		{
			name: "valid integer array input (cache hit)",
			request: map[string]interface{}{
				"input": []int{1, 2, 3, 4},
				"model": "text-embedding-ada-002",
			},
			authHeader:   "Bearer " + validAPIKey,
			wantStatus:   http.StatusOK,
			wantCacheHit: true,
			wantTokens:   0,
		},
		{
			name: "valid float array input",
			request: map[string]interface{}{
				"input": []float64{1.5, 2.5, 3.5, 4.5},
				"model": "text-embedding-ada-002",
			},
			authHeader:   "Bearer " + validAPIKey,
			wantStatus:   http.StatusOK,
			wantTokens:   8,
			wantCacheHit: false,
		},
		{
			name: "valid float array input (cache hit)",
			request: map[string]interface{}{
				"input": []float64{1.5, 2.5, 3.5, 4.5},
				"model": "text-embedding-ada-002",
			},
			authHeader:   "Bearer " + validAPIKey,
			wantStatus:   http.StatusOK,
			wantCacheHit: true,
			wantTokens:   0,
		},
		{
			name: "invalid array element type",
			request: map[string]interface{}{
				"input": []interface{}{"string", 1, 2.0},
				"model": "text-embedding-ada-002",
			},
			authHeader:    "Bearer " + validAPIKey,
			wantStatus:    http.StatusBadRequest,
			wantErrorType: "invalid_request_error",
		},
		{
			name: "valid request - initial (cache miss)",
			request: map[string]interface{}{
				"input": "Hello, World!",
				"model": "text-embedding-ada-002",
			},
			authHeader:   "Bearer " + validAPIKey,
			wantStatus:   http.StatusOK,
			wantCacheHit: false,
			wantTokens:   8,
		},
		{
			name: "valid request - cached (cache hit)",
			request: map[string]interface{}{
				"input": "Hello, World!",
				"model": "text-embedding-ada-002",
			},
			authHeader:   "Bearer " + validAPIKey,
			wantStatus:   http.StatusOK,
			wantCacheHit: true,
			wantTokens:   0,
		},
		{
			name:       "invalid method",
			request:    map[string]interface{}{},
			wantStatus: http.StatusMethodNotAllowed,
		},
		{
			name:       "invalid path",
			request:    map[string]interface{}{},
			wantStatus: http.StatusNotFound,
		},
		{
			name: "missing auth header",
			request: map[string]interface{}{
				"input": "test",
				"model": "text-embedding-ada-002",
			},
			wantStatus:    http.StatusUnauthorized,
			wantErrorType: "invalid_request_error",
		},
		{
			name: "invalid auth format",
			request: map[string]interface{}{
				"input": "test",
				"model": "text-embedding-ada-002",
			},
			authHeader:    "Invalid " + validAPIKey,
			wantStatus:    http.StatusUnauthorized,
			wantErrorType: "invalid_request_error",
		},
		{
			name: "invalid api key",
			request: map[string]interface{}{
				"input": "test",
				"model": "text-embedding-ada-002",
			},
			authHeader:    "Bearer invalid-key",
			wantStatus:    http.StatusUnauthorized,
			wantErrorType: "invalid_request_error",
		},
		{
			name: "invalid model",
			request: map[string]interface{}{
				"input": "test",
				"model": "invalid-model",
			},
			authHeader:    "Bearer " + validAPIKey,
			wantStatus:    http.StatusBadRequest,
			wantErrorType: "invalid_request_error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := NewHandler(allowedModels, apiKeyPattern, ts.URL, db, false)

			var body []byte
			var err error
			body, err = json.Marshal(tt.request)
			if err != nil {
				t.Fatalf("Failed to marshal request body: %v", err)
			}

			method := "POST"
			path := "/v1/embeddings"
			if tt.name == "invalid method" {
				method = "GET"
			}
			if tt.name == "invalid path" {
				path = "/v1/invalid"
			}

			req := httptest.NewRequest(method, path, bytes.NewReader(body))
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			handler.ServeHTTP(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("Expected status code %d, got %d", tt.wantStatus, w.Code)
			}

			if tt.wantStatus == http.StatusOK {
				var resp upstream.EmbeddingResponse
				if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
					t.Fatalf("Failed to decode response: %v", err)
				}

				// キャッシュヒットの検証
				if tt.wantCacheHit && resp.Usage.TotalTokens != 0 {
					t.Error("Expected cache hit (zero tokens), got cache miss")
				}
				if !tt.wantCacheHit && resp.Usage.TotalTokens != tt.wantTokens {
					t.Errorf("Expected %d tokens, got %d", tt.wantTokens, resp.Usage.TotalTokens)
				}

				// レスポンスの基本的な検証
				if len(resp.Data) != 1 {
					t.Errorf("Expected 1 embedding, got %d", len(resp.Data))
					return
				}
				// 型アサーションの前にnilチェック
				if resp.Data[0].Embedding == nil {
					t.Error("Embedding is nil")
					return
				}

				// []interface{}として処理
				switch embedding := resp.Data[0].Embedding.(type) {
				case []interface{}:
					if len(embedding) != 3 {
						t.Errorf("Expected embedding length 3, got %d", len(embedding))
						return
					}
					// 各要素が数値であることを確認
					for i, v := range embedding {
						switch v.(type) {
						case float64, float32:
							// OK
						default:
							t.Errorf("Expected float at index %d, got %T", i, v)
						}
					}
				case []float64:
					if len(embedding) != 3 {
						t.Errorf("Expected embedding length 3, got %d", len(embedding))
					}
				case []float32:
					if len(embedding) != 3 {
						t.Errorf("Expected embedding length 3, got %d", len(embedding))
					}
				default:
					t.Errorf("Expected embedding array, got %T", resp.Data[0].Embedding)
				}
			}
		})
	}
}

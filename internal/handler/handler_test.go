package handler

import (
	"bytes"
	"encoding/json"
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
		resp := upstream.EmbeddingResponse{
			Object: "list",
			Data: []struct {
				Object    string    `json:"object"`
				Embedding []float32 `json:"embedding"`
				Index     int       `json:"index"`
			}{
				{
					Object:    "embedding",
					Embedding: []float32{0.1, 0.2, 0.3},
					Index:     0,
				},
			},
			Model: "text-embedding-ada-002",
			Usage: struct {
				PromptTokens int `json:"prompt_tokens"`
				TotalTokens  int `json:"total_tokens"`
			}{
				PromptTokens: 8,
				TotalTokens:  8,
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer ts.Close()

	allowedModels := []string{"text-embedding-ada-002"}
	apiKeyPattern := "^sk-[a-zA-Z0-9]{32}$"
	validAPIKey := "sk-abcdefghijklmnopqrstuvwxyz123456" // 有効なAPIキーの例

	tests := []struct {
		name         string
		method       string
		path         string
		body         *upstream.EmbeddingRequest
		authHeader   string
		wantStatus   int
		wantCacheHit bool // キャッシュヒットを期待するかどうか
		wantTokens   int  // 期待されるトークン数（キャッシュヒット時は0）
	}{
		{
			name:   "valid request - initial (cache miss)",
			method: "POST",
			path:   "/v1/embeddings",
			body: &upstream.EmbeddingRequest{
				Input: "Hello, World!",
				Model: "text-embedding-ada-002",
			},
			authHeader:   "Bearer " + validAPIKey,
			wantStatus:   http.StatusOK,
			wantCacheHit: false,
			wantTokens:   8,
		},
		{
			name:   "valid request - cached (cache hit)",
			method: "POST",
			path:   "/v1/embeddings",
			body: &upstream.EmbeddingRequest{
				Input: "Hello, World!",
				Model: "text-embedding-ada-002",
			},
			authHeader:   "Bearer " + validAPIKey,
			wantStatus:   http.StatusOK,
			wantCacheHit: true,
			wantTokens:   0,
		},
		{
			name:       "invalid method",
			method:     "GET",
			path:       "/v1/embeddings",
			wantStatus: http.StatusMethodNotAllowed,
		},
		{
			name:       "invalid path",
			method:     "POST",
			path:       "/v1/invalid",
			wantStatus: http.StatusNotFound,
		},
		{
			name:   "missing auth header",
			method: "POST",
			path:   "/v1/embeddings",
			body: &upstream.EmbeddingRequest{
				Input: "test",
				Model: "text-embedding-ada-002",
			},
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:   "invalid auth format",
			method: "POST",
			path:   "/v1/embeddings",
			body: &upstream.EmbeddingRequest{
				Input: "test",
				Model: "text-embedding-ada-002",
			},
			authHeader: "Invalid " + validAPIKey,
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:   "invalid api key",
			method: "POST",
			path:   "/v1/embeddings",
			body: &upstream.EmbeddingRequest{
				Input: "test",
				Model: "text-embedding-ada-002",
			},
			authHeader: "Bearer invalid-key",
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:   "invalid model",
			method: "POST",
			path:   "/v1/embeddings",
			body: &upstream.EmbeddingRequest{
				Input: "test",
				Model: "invalid-model",
			},
			authHeader: "Bearer " + validAPIKey,
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := NewHandler(allowedModels, apiKeyPattern, ts.URL, db, false)

			var body []byte
			if tt.body != nil {
				var err error
				body, err = json.Marshal(tt.body)
				if err != nil {
					t.Fatalf("Failed to marshal request body: %v", err)
				}
			}

			req := httptest.NewRequest(tt.method, tt.path, bytes.NewReader(body))
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
				}
				if len(resp.Data[0].Embedding) != 3 {
					t.Errorf("Expected embedding length 3, got %d", len(resp.Data[0].Embedding))
				}
			}
		})
	}
}

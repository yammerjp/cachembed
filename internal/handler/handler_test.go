package handler

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/yammerjp/cachembed/internal/storage"
	"github.com/yammerjp/cachembed/internal/upstream"
)

func TestHandleRequest(t *testing.T) {
	// モックレスポンスを生成する関数
	createMockHandler := func() http.HandlerFunc {
		requestCount := 0 // クロージャで状態を保持
		return func(w http.ResponseWriter, r *http.Request) {
			requestCount++
			// 2回目以降のリクエストはキャッシュから取得されるはずなので、
			// このハンドラーは1回目のリクエストでのみ呼ばれるはず
			if requestCount > 1 {
				t.Error("Unexpected request to upstream server")
				return
			}

			// リクエストの検証
			if r.Header.Get("Authorization") != "Bearer test-key" {
				t.Error("Authorization header mismatch")
			}
			if r.Header.Get("Content-Type") != "application/json" {
				t.Error("Content-Type header mismatch")
			}

			// リクエストボディの検証
			body, _ := io.ReadAll(r.Body)
			var req map[string]interface{}
			if err := json.Unmarshal(body, &req); err != nil {
				t.Errorf("Failed to parse request body: %v", err)
			}
			if req["input"] != "Hello, world" {
				t.Errorf("Input mismatch: got %v", req["input"])
			}

			// APIレスポンスを作成
			resp := upstream.EmbeddingResponse{
				Object: "list",
				Data: []upstream.EmbeddingData{
					{
						Object:    "embedding",
						Embedding: []float32{0.1, 0.2, 0.3},
						Index:     0,
					},
				},
				Model: "text-embedding-ada-002",
				Usage: upstream.Usage{
					PromptTokens: 8,
					TotalTokens:  8,
				},
			}

			// レスポンスを返す
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)
		}
	}

	tests := []struct {
		name    string
		request map[string]interface{}
		setup   func(t *testing.T, server *httptest.Server, db storage.Database)
	}{
		{
			name: "successful single string input",
			request: map[string]interface{}{
				"input": "Hello, world",
				"model": "text-embedding-ada-002",
			},
			setup: func(t *testing.T, server *httptest.Server, db storage.Database) {
				server.Config.Handler = createMockHandler()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// テスト用のデータベースを作成
			db, err := storage.NewDB(":memory:")
			if err != nil {
				t.Fatalf("Failed to create test database: %v", err)
			}
			defer db.Close()

			// モックサーバーの設定
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
			defer server.Close()

			// テストケース固有のセットアップを実行
			tt.setup(t, server, db)

			// テスト用のハンドラを作成
			h := NewHandler(
				[]string{"text-embedding-ada-002"},
				nil, // API keyの正規表現チェックは無効
				db,
				upstream.NewClient(server.Client(), server.URL),
				false,
			)

			// リクエストJSONの作成
			reqJSON, err := json.Marshal(tt.request)
			if err != nil {
				t.Fatalf("Failed to marshal request: %v", err)
			}

			// 1回目のリクエスト
			req := httptest.NewRequest(http.MethodPost, "/v1/embeddings", bytes.NewReader(reqJSON))
			req.Header.Set("Authorization", "Bearer test-key")
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			err = h.handleRequest(w, req)
			if err != nil {
				t.Fatalf("First request failed: %v", err)
			}

			if w.Code != http.StatusOK {
				t.Errorf("Expected status code %d, got %d", http.StatusOK, w.Code)
			}

			var resp upstream.EmbeddingResponse
			if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
				t.Fatalf("Failed to decode response: %v", err)
			}

			// レスポンスの検証
			if len(resp.Data) != 1 {
				t.Errorf("Expected 1 embedding, got %d", len(resp.Data))
			}
			if resp.Usage.PromptTokens != 8 {
				t.Errorf("Expected 8 prompt tokens, got %d", resp.Usage.PromptTokens)
			}
			if resp.Usage.TotalTokens != 8 {
				t.Errorf("Expected 8 total tokens, got %d", resp.Usage.TotalTokens)
			}

			// 2回目のリクエスト（キャッシュから取得されるはず）
			req = httptest.NewRequest(http.MethodPost, "/v1/embeddings", bytes.NewReader(reqJSON))
			req.Header.Set("Authorization", "Bearer test-key")
			req.Header.Set("Content-Type", "application/json")
			w2 := httptest.NewRecorder()

			err = h.handleRequest(w2, req)
			if err != nil {
				t.Fatalf("Second request failed: %v", err)
			}

			var resp2 upstream.EmbeddingResponse
			if err := json.NewDecoder(w2.Body).Decode(&resp2); err != nil {
				t.Fatalf("Failed to decode second response: %v", err)
			}

			// キャッシュされたレスポンスの検証
			if resp2.Usage.PromptTokens != 0 {
				t.Errorf("Expected 0 prompt tokens for cached response, got %d", resp2.Usage.PromptTokens)
			}
			if resp2.Usage.TotalTokens != 0 {
				t.Errorf("Expected 0 total tokens for cached response, got %d", resp2.Usage.TotalTokens)
			}
		})
	}
}

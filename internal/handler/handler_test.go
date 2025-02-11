package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/yammerjp/cachembed/internal/storage"
	"github.com/yammerjp/cachembed/internal/upstream"
)

func TestHandleRequest(t *testing.T) {
	type InitialData struct {
		inputHash string
		model     string
		embedding []float32
	}

	type Request struct {
		Input          interface{} `json:"input"`
		Model          string      `json:"model"`
		EncodingFormat string      `json:"encoding_format,omitempty"`
	}

	type mockUpstream struct {
		expectedInput  interface{}
		expectedFormat string
		mockResponse   *upstream.EmbeddingResponse
		callCount      *int
	}

	type expectedResponse struct {
		status         int
		body           *upstream.EmbeddingResponse
		errorType      string
		errorMsg       string
		encodingFormat string
	}

	tests := []struct {
		name             string
		initialData      []InitialData
		request          Request
		mockUpstream     mockUpstream
		expectedResponse expectedResponse
	}{
		{
			name: "successful single string input",
			request: Request{
				Input: "Hello, world",
				Model: "text-embedding-ada-002",
			},
			mockUpstream: mockUpstream{
				expectedInput:  "Hello, world",
				expectedFormat: "base64",
				mockResponse: &upstream.EmbeddingResponse{
					Object: "list",
					Data: []upstream.EmbeddingData{
						{
							Object:    "embedding",
							Embedding: dummyVecBase64Str,
							Index:     0,
						},
					},
					Model: "text-embedding-ada-002",
					Usage: upstream.Usage{
						PromptTokens: 8,
						TotalTokens:  8,
					},
				},
			},
			expectedResponse: expectedResponse{
				status:         http.StatusOK,
				encodingFormat: "",
				body: &upstream.EmbeddingResponse{
					Object: "list",
					Data: []upstream.EmbeddingData{
						{
							Object:    "embedding",
							Embedding: dummyVec,
							Index:     0,
						},
					},
					Model: "text-embedding-ada-002",
					Usage: upstream.Usage{
						PromptTokens: 8,
						TotalTokens:  8,
					},
				},
			},
		},
		{
			name: "invalid model error",
			request: Request{
				Input: "Hello, world",
				Model: "invalid-model",
			},
			mockUpstream: mockUpstream{},
			expectedResponse: expectedResponse{
				status:    http.StatusBadRequest,
				errorType: "invalid_request_error",
				errorMsg:  "Unsupported model: invalid-model",
			},
		},
		{
			name: "base64 encoding format",
			request: Request{
				Input:          "Hello, world",
				Model:          "text-embedding-ada-002",
				EncodingFormat: "base64",
			},
			mockUpstream: mockUpstream{
				expectedInput:  "Hello, world",
				expectedFormat: "base64",
				mockResponse: &upstream.EmbeddingResponse{
					Object: "list",
					Data: []upstream.EmbeddingData{
						{
							Object:    "embedding",
							Embedding: dummyVecBase64Str,
							Index:     0,
						},
					},
					Model: "text-embedding-ada-002",
					Usage: upstream.Usage{
						PromptTokens: 8,
						TotalTokens:  8,
					},
				},
			},
			expectedResponse: expectedResponse{
				status:         http.StatusOK,
				encodingFormat: "base64",
				body: &upstream.EmbeddingResponse{
					Object: "list",
					Data: []upstream.EmbeddingData{
						{
							Object:    "embedding",
							Embedding: dummyVecBase64Str,
							Index:     0,
						},
					},
					Model: "text-embedding-ada-002",
					Usage: upstream.Usage{
						PromptTokens: 8,
						TotalTokens:  8,
					},
				},
			},
		},
		{
			name: "cache hit",
			initialData: []InitialData{
				{
					inputHash: "e02aa1b106d5c7c6a98def2b13005d5b84fd8dc8",
					model:     "text-embedding-ada-002",
					embedding: []float32{0.1, 0.2, 0.3},
				},
			},
			request: Request{
				Input: "Hello, world",
				Model: "text-embedding-ada-002",
			},
			mockUpstream: mockUpstream{
				callCount: new(int),
			},
			expectedResponse: expectedResponse{
				status: http.StatusOK,
				body: &upstream.EmbeddingResponse{
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
						PromptTokens: 0,
						TotalTokens:  0,
					},
				},
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

			// マイグレーションを実行
			if err := db.RunMigrations(); err != nil {
				t.Fatalf("Failed to run migrations: %v", err)
			}

			// 初期データをロード
			for _, data := range tt.initialData {
				if err := db.StoreEmbedding(data.inputHash, data.model, data.embedding); err != nil {
					t.Fatalf("Failed to store initial data: %v", err)
				}
			}

			// モックサーバーの設定
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// upstreamの呼び出し回数をインクリメント
				if tt.mockUpstream.callCount != nil {
					*tt.mockUpstream.callCount++
				}

				// invalid modelの場合は早期リターン
				if tt.expectedResponse.status == http.StatusBadRequest {
					w.WriteHeader(http.StatusBadRequest)
					json.NewEncoder(w).Encode(map[string]interface{}{
						"error": map[string]interface{}{
							"message": tt.expectedResponse.errorMsg,
							"type":    tt.expectedResponse.errorType,
						},
					})
					return
				}

				var req map[string]interface{}
				if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
					t.Errorf("Failed to parse request body: %v", err)
					return
				}

				// input値の検証
				if !reflect.DeepEqual(req["input"], tt.mockUpstream.expectedInput) {
					t.Errorf("Input mismatch: got %v, want %v", req["input"], tt.mockUpstream.expectedInput)
				}

				// encoding_formatの検証
				format, ok := req["encoding_format"].(string)
				if tt.mockUpstream.expectedFormat != "" {
					if !ok || format != tt.mockUpstream.expectedFormat {
						t.Errorf("Encoding format mismatch: got %v, want %v", format, tt.mockUpstream.expectedFormat)
					}
				}

				// モックレスポンスを返す
				w.Header().Set("Content-Type", "application/json")
				if err := json.NewEncoder(w).Encode(tt.mockUpstream.mockResponse); err != nil {
					t.Errorf("Failed to encode mock response: %v", err)
				}
			}))
			defer server.Close()

			// テスト用のハンドラを作成
			h := NewHandler(
				[]string{"text-embedding-ada-002"},
				nil,
				db,
				upstream.NewClient(server.Client(), server.URL),
				false,
			)

			// リクエストの実行
			reqJSON, err := json.Marshal(tt.request)
			if err != nil {
				t.Fatalf("Failed to marshal request: %v", err)
			}

			req := httptest.NewRequest(http.MethodPost, "/v1/embeddings", bytes.NewReader(reqJSON))
			req.Header.Set("Authorization", "Bearer test-key")
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			err = h.handleRequest(w, req)

			// ステータスコードの検証
			if w.Code != tt.expectedResponse.status {
				t.Errorf("Expected status code %d, got %d", tt.expectedResponse.status, w.Code)
			}

			// キャッシュヒットの場合、upstreamが呼ばれていないことを確認
			if tt.name == "cache hit" && *tt.mockUpstream.callCount > 0 {
				t.Errorf("Expected no upstream calls for cache hit, got %d calls", *tt.mockUpstream.callCount)
			}

			// レスポンスボディの検証
			if tt.expectedResponse.body != nil {
				var resp upstream.EmbeddingResponse
				if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
					t.Fatalf("Failed to decode response: %v", err)
				}

				// JSONに変換して比較
				gotJSON, err := json.Marshal(resp)
				if err != nil {
					t.Fatalf("Failed to marshal response: %v", err)
				}
				wantJSON, err := json.Marshal(tt.expectedResponse.body)
				if err != nil {
					t.Fatalf("Failed to marshal expected response: %v", err)
				}

				// 比較のために一度JSONをマップに変換
				var got, want map[string]interface{}
				if err := json.Unmarshal(gotJSON, &got); err != nil {
					t.Fatalf("Failed to unmarshal response JSON: %v", err)
				}
				if err := json.Unmarshal(wantJSON, &want); err != nil {
					t.Fatalf("Failed to unmarshal expected JSON: %v", err)
				}

				if !reflect.DeepEqual(got, want) {
					t.Errorf("Response mismatch:\ngot: %s\nwant: %s", gotJSON, wantJSON)
				}
			} else {
				var errorResp struct {
					Error struct {
						Message string `json:"message"`
						Type    string `json:"type"`
					} `json:"error"`
				}
				if err := json.NewDecoder(w.Body).Decode(&errorResp); err != nil {
					t.Fatalf("Failed to decode error response: %v", err)
				}
				if errorResp.Error.Type != tt.expectedResponse.errorType {
					t.Errorf("Error type mismatch: got %v, want %v", errorResp.Error.Type, tt.expectedResponse.errorType)
				}
				if errorResp.Error.Message != tt.expectedResponse.errorMsg {
					t.Errorf("Error message mismatch: got %v, want %v", errorResp.Error.Message, tt.expectedResponse.errorMsg)
				}
			}
		})
	}
}

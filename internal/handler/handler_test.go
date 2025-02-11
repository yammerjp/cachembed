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
	"github.com/yammerjp/cachembed/internal/util"
)

func TestHandleRequest(t *testing.T) {
	type InitialData struct {
		inputHash string
		model     string
		embedding util.EmbeddedVectorBase64
		dimension int
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
							Embedding: util.Base64Dummy1,
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
							Embedding: util.VecDummy1,
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
							Embedding: util.Base64Dummy1,
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
							Embedding: util.Base64Dummy1,
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
					embedding: util.Base64Dummy1,
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
							Embedding: util.VecDummy1,
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
		{
			name: "token sequence input",
			request: Request{
				Input: []int{1, 2, 3, 4},
				Model: "text-embedding-ada-002",
			},
			mockUpstream: mockUpstream{
				expectedInput:  []interface{}{float64(1), float64(2), float64(3), float64(4)},
				expectedFormat: "base64",
				mockResponse: &upstream.EmbeddingResponse{
					Object: "list",
					Data: []upstream.EmbeddingData{
						{
							Object:    "embedding",
							Embedding: util.Base64Dummy1,
							Index:     0,
						},
					},
					Model: "text-embedding-ada-002",
					Usage: upstream.Usage{
						PromptTokens: 4,
						TotalTokens:  4,
					},
				},
				callCount: new(int),
			},
			expectedResponse: expectedResponse{
				status: http.StatusOK,
				body: &upstream.EmbeddingResponse{
					Object: "list",
					Data: []upstream.EmbeddingData{
						{
							Object:    "embedding",
							Embedding: util.VecDummy1,
							Index:     0,
						},
					},
					Model: "text-embedding-ada-002",
					Usage: upstream.Usage{
						PromptTokens: 4,
						TotalTokens:  4,
					},
				},
			},
		},
		{
			name: "string array input",
			request: Request{
				Input:          []string{"Hello", "World"},
				Model:          "text-embedding-ada-002",
				EncodingFormat: "base64",
			},
			mockUpstream: mockUpstream{
				expectedInput:  []interface{}{"Hello", "World"},
				expectedFormat: "base64",
				mockResponse: &upstream.EmbeddingResponse{
					Object: "list",
					Data: []upstream.EmbeddingData{
						{
							Object:    "embedding",
							Embedding: util.Base64Dummy1,
							Index:     0,
						},
						{
							Object:    "embedding",
							Embedding: util.Base64Dummy2,
							Index:     1,
						},
					},
					Model: "text-embedding-ada-002",
					Usage: upstream.Usage{
						PromptTokens: 2,
						TotalTokens:  2,
					},
				},
				callCount: new(int),
			},
			expectedResponse: expectedResponse{
				status:         http.StatusOK,
				encodingFormat: "base64",
				body: &upstream.EmbeddingResponse{
					Object: "list",
					Data: []upstream.EmbeddingData{
						{
							Object:    "embedding",
							Embedding: util.Base64Dummy1,
							Index:     0,
						},
						{
							Object:    "embedding",
							Embedding: util.Base64Dummy2,
							Index:     1,
						},
					},
					Model: "text-embedding-ada-002",
					Usage: upstream.Usage{
						PromptTokens: 2,
						TotalTokens:  2,
					},
				},
			},
		},
		{
			name: "partial_cache_hit_with_string_array",
			initialData: []InitialData{
				{
					inputHash: "70c07ec18ef89c5309bbb0937f3a6342411e1fdd", // "World"のハッシュ
					model:     "text-embedding-ada-002",
					embedding: util.Base64Dummy1,
				},
			},
			request: Request{
				Input: []string{"Hello", "World"},
				Model: "text-embedding-ada-002",
			},
			mockUpstream: mockUpstream{
				expectedInput:  []interface{}{"Hello"}, // キャッシュミスの部分のみupstreamに送信
				expectedFormat: "base64",
				mockResponse: &upstream.EmbeddingResponse{
					Object: "list",
					Data: []upstream.EmbeddingData{
						{
							Object:    "embedding",
							Embedding: util.Base64Dummy1,
							Index:     0,
						},
					},
					Model: "text-embedding-ada-002",
					Usage: upstream.Usage{
						PromptTokens: 1,
						TotalTokens:  1,
					},
				},
				callCount: new(int),
			},
			expectedResponse: expectedResponse{
				status: http.StatusOK,
				body: &upstream.EmbeddingResponse{
					Object: "list",
					Data: []upstream.EmbeddingData{
						{
							Object:    "embedding",
							Embedding: util.VecDummy1,
							Index:     0,
						},
						{
							Object:    "embedding",
							Embedding: []float32{0.125, 0.25, 0.5},
							Index:     1,
						},
					},
					Model: "text-embedding-ada-002",
					Usage: upstream.Usage{
						PromptTokens: 1,
						TotalTokens:  1,
					},
				},
			},
		},
		{
			name: "token_sequences_input_with_partial_cache",
			initialData: []InitialData{
				{
					inputHash: "ef510977d15a2498e88750ef53092291499b5e83", // [3, 4]
					model:     "text-embedding-ada-002",
					embedding: util.Base64Dummy2,
				},
				{
					inputHash: "b8b9b72a24e33fac417a60f2064434b95d408eda", // [7, 8]
					model:     "text-embedding-ada-002",
					embedding: util.Base64Dummy4,
				},
			},
			request: Request{
				Input: [][]int{{1, 2}, {3, 4}, {5, 6}, {7, 8}},
				Model: "text-embedding-ada-002",
			},
			mockUpstream: mockUpstream{
				expectedInput: []interface{}{
					[]interface{}{float64(1), float64(2)},
					[]interface{}{float64(5), float64(6)},
				},
				expectedFormat: "base64",
				mockResponse: &upstream.EmbeddingResponse{
					Object: "list",
					Data: []upstream.EmbeddingData{
						{
							Object:    "embedding",
							Embedding: util.Base64Dummy1,
							Index:     0,
						},
						{
							Object:    "embedding",
							Embedding: util.Base64Dummy3,
							Index:     1,
						},
					},
					Model: "text-embedding-ada-002",
					Usage: upstream.Usage{
						PromptTokens: 4,
						TotalTokens:  4,
					},
				},
				callCount: new(int),
			},
			expectedResponse: expectedResponse{
				status: http.StatusOK,
				body: &upstream.EmbeddingResponse{
					Object: "list",
					Data: []upstream.EmbeddingData{
						{
							Object:    "embedding",
							Embedding: util.VecDummy1,
							Index:     0,
						},
						{
							Object:    "embedding",
							Embedding: util.VecDummy2,
							Index:     1,
						},
						{
							Object:    "embedding",
							Embedding: util.VecDummy3,
							Index:     2,
						},
						{
							Object:    "embedding",
							Embedding: util.VecDummy4,
							Index:     3,
						},
					},
					Model: "text-embedding-ada-002",
					Usage: upstream.Usage{
						PromptTokens: 4,
						TotalTokens:  4,
					},
				},
			},
		},
		{
			name: "cache miss with different model",
			initialData: []InitialData{
				{
					inputHash: "e02aa1b106d5c7c6a98def2b13005d5b84fd8dc8", // "Hello, world"のハッシュ
					model:     "text-embedding-3-small",
					embedding: util.Base64Dummy1,
				},
			},
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
							Embedding: util.Base64Dummy1,
							Index:     0,
						},
					},
					Model: "text-embedding-ada-002",
					Usage: upstream.Usage{
						PromptTokens: 8,
						TotalTokens:  8,
					},
				},
				callCount: new(int),
			},
			expectedResponse: expectedResponse{
				status: http.StatusOK,
				body: &upstream.EmbeddingResponse{
					Object: "list",
					Data: []upstream.EmbeddingData{
						{
							Object:    "embedding",
							Embedding: util.VecDummy1,
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
				if err := db.StoreEmbedding(data.inputHash, data.model, data.dimension, util.EmbeddedVectorBase64(data.embedding)); err != nil {
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

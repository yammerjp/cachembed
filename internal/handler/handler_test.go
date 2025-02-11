package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"regexp"
	"strings"
	"testing"

	"github.com/yammerjp/cachembed/internal/storage"
	"github.com/yammerjp/cachembed/internal/upstream"
)

func TestHandleEmbeddings(t *testing.T) {
	validAPIKey := "test-api-key"
	mockDB := storage.NewMockDB()
	mockUpstream := upstream.NewMockClient()

	// 正規表現のコンパイル
	apiKeyRegexp := regexp.MustCompile("^test-.*$")

	// テストケース共通の埋め込みベクトル
	mockEmbedding := []float32{0.1, 0.2, 0.3}
	mockBase64 := float32ToBase64(mockEmbedding)

	tests := []struct {
		name          string
		request       map[string]interface{}
		encoding      string // encoding_formatの指定（空文字列="float"）
		setupCache    bool   // テスト前にキャッシュを設定するか
		authHeader    string
		wantStatus    int
		wantTokens    int         // キャッシュミス時の期待するトークン数
		wantCacheHit  bool        // キャッシュヒットを期待するか
		wantEmbedding interface{} // 期待する埋め込みベクトルの形式
		wantError     string
	}{
		// 1. 単一文字列入力のテスト
		{
			name: "single string - cache miss - float",
			request: map[string]interface{}{
				"input": "Hello, World!",
				"model": "text-embedding-3-small",
			},
			wantStatus:    http.StatusOK,
			wantTokens:    3,
			wantCacheHit:  false,
			wantEmbedding: mockEmbedding,
		},
		{
			name: "single string - cache hit - float",
			request: map[string]interface{}{
				"input": "Hello, World!",
				"model": "text-embedding-3-small",
			},
			setupCache:    true,
			wantStatus:    http.StatusOK,
			wantCacheHit:  true,
			wantEmbedding: mockEmbedding,
		},
		{
			name: "single string - cache miss - base64",
			request: map[string]interface{}{
				"input": "Hello, World!",
				"model": "text-embedding-3-small",
			},
			encoding:      "base64",
			wantStatus:    http.StatusOK,
			wantTokens:    3,
			wantCacheHit:  false,
			wantEmbedding: mockBase64,
		},

		// 2. トークン列（整数配列）入力のテスト
		{
			name: "token array - cache miss - float",
			request: map[string]interface{}{
				"input": []interface{}{1, 2, 3},
				"model": "text-embedding-3-small",
			},
			wantStatus:    http.StatusOK,
			wantTokens:    3,
			wantCacheHit:  false,
			wantEmbedding: mockEmbedding,
		},
		{
			name: "token array - cache hit - base64",
			request: map[string]interface{}{
				"input": []interface{}{1, 2, 3},
				"model": "text-embedding-3-small",
			},
			encoding:      "base64",
			setupCache:    true,
			wantStatus:    http.StatusOK,
			wantCacheHit:  true,
			wantEmbedding: mockBase64,
		},
		{
			name: "token array with float - error",
			request: map[string]interface{}{
				"input": []interface{}{1.5, 2, 3},
				"model": "text-embedding-3-small",
			},
			wantStatus: http.StatusBadRequest,
			wantError:  "non-integer number at index 0: 1.5",
		},

		// 3. 文字列配列入力のテスト
		{
			name: "string array - cache miss - float",
			request: map[string]interface{}{
				"input": []interface{}{"Hello", "World"},
				"model": "text-embedding-3-small",
			},
			wantStatus:    http.StatusOK,
			wantTokens:    2,
			wantCacheHit:  false,
			wantEmbedding: mockEmbedding,
		},
		{
			name: "string array - cache hit - base64",
			request: map[string]interface{}{
				"input": []interface{}{"Hello", "World"},
				"model": "text-embedding-3-small",
			},
			encoding:      "base64",
			setupCache:    true,
			wantStatus:    http.StatusOK,
			wantCacheHit:  true,
			wantEmbedding: mockBase64,
		},

		// 4. トークン列の配列入力のテスト
		{
			name: "token arrays - cache miss - float",
			request: map[string]interface{}{
				"input": []interface{}{
					[]interface{}{1, 2},
					[]interface{}{3, 4},
				},
				"model": "text-embedding-3-small",
			},
			wantStatus:    http.StatusOK,
			wantTokens:    4,
			wantCacheHit:  false,
			wantEmbedding: mockEmbedding,
		},
		{
			name: "token arrays - cache hit - base64",
			request: map[string]interface{}{
				"input": []interface{}{
					[]interface{}{1, 2},
					[]interface{}{3, 4},
				},
				"model": "text-embedding-3-small",
			},
			encoding:      "base64",
			setupCache:    true,
			wantStatus:    http.StatusOK,
			wantCacheHit:  true,
			wantEmbedding: mockBase64,
		},
		{
			name: "token arrays with float - error",
			request: map[string]interface{}{
				"input": []interface{}{
					[]interface{}{1.5, 2},
					[]interface{}{3, 4},
				},
				"model": "text-embedding-3-small",
			},
			wantStatus: http.StatusBadRequest,
			wantError:  "non-integer number in array at index 0,0: 1.5",
		},

		// 5. エラーケース
		{
			name: "empty input array",
			request: map[string]interface{}{
				"input": []interface{}{},
				"model": "text-embedding-3-small",
			},
			wantStatus: http.StatusBadRequest,
			wantError:  "input array must not be empty",
		},
		{
			name: "invalid input type",
			request: map[string]interface{}{
				"input": true,
				"model": "text-embedding-3-small",
			},
			wantStatus: http.StatusBadRequest,
			wantError:  "invalid input type: got bool",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// テスト用ハンドラーの準備
			h := NewHandler(
				[]string{"text-embedding-3-small"},
				apiKeyRegexp,
				mockDB,
				mockUpstream,
				false,
			)

			// キャッシュの設定
			if tt.setupCache {
				input := tt.request["input"]
				model := tt.request["model"].(string)
				hash, err := calculateInputHash(input)
				if err != nil {
					t.Fatalf("failed to calculate hash: %v", err)
				}
				err = mockDB.StoreEmbedding(hash, model, mockEmbedding)
				if err != nil {
					t.Fatalf("failed to setup cache: %v", err)
				}
			}

			// リクエストの準備
			if tt.encoding != "" {
				tt.request["encoding_format"] = tt.encoding
			}
			reqBody, err := json.Marshal(tt.request)
			if err != nil {
				t.Fatalf("failed to marshal request: %v", err)
			}

			req := httptest.NewRequest(http.MethodPost, "/v1/embeddings", bytes.NewReader(reqBody))
			req.Header.Set("Authorization", "Bearer "+validAPIKey)
			req.Header.Set("Content-Type", "application/json")

			// レスポンスの取得
			w := httptest.NewRecorder()
			h.ServeHTTP(w, req)

			// ステータスコードの検証
			if w.Code != tt.wantStatus {
				t.Errorf("status code = %d, want %d", w.Code, tt.wantStatus)
			}

			// エラーケースの検証
			if tt.wantError != "" {
				var errResp struct {
					Error struct {
						Message string `json:"message"`
					} `json:"error"`
				}
				if err := json.NewDecoder(w.Body).Decode(&errResp); err != nil {
					t.Fatalf("failed to decode error response: %v", err)
				}
				if !strings.Contains(errResp.Error.Message, tt.wantError) {
					t.Errorf("error message = %q, want to contain %q", errResp.Error.Message, tt.wantError)
				}
				return
			}

			// 成功ケースのレスポンス検証
			var resp upstream.EmbeddingResponse
			if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
				t.Fatalf("failed to decode response: %v", err)
			}

			// キャッシュヒット/ミスの検証
			if tt.wantCacheHit && resp.Usage.TotalTokens != 0 {
				t.Error("Expected cache hit (zero tokens), got cache miss")
			}
			if !tt.wantCacheHit && resp.Usage.TotalTokens != tt.wantTokens {
				t.Errorf("Expected %d tokens, got %d", tt.wantTokens, resp.Usage.TotalTokens)
			}

			// 埋め込みベクトルの検証
			if len(resp.Data) == 0 {
				t.Fatal("no embeddings in response")
			}
			embedding := resp.Data[0].Embedding
			if !reflect.DeepEqual(embedding, tt.wantEmbedding) {
				t.Errorf("embedding = %v, want %v", embedding, tt.wantEmbedding)
			}
		})
	}
}

// テスト用のヘルパー関数
func calculateInputHash(input interface{}) (string, error) {
	var hashInput string

	switch v := input.(type) {
	case string:
		hashInput = v
	case []interface{}:
		if len(v) == 0 {
			return "", fmt.Errorf("input array must not be empty")
		}

		// 最初の要素が配列かどうかをチェック
		if _, isArray := v[0].([]interface{}); isArray {
			// 2次元配列の処理
			parts := make([]string, len(v))
			for i, arr := range v {
				subArr, ok := arr.([]interface{})
				if !ok {
					return "", fmt.Errorf("invalid array element at index %d", i)
				}
				if len(subArr) == 0 {
					return "", fmt.Errorf("empty sub-array at index %d", i)
				}

				// サブ配列の処理
				subParts := make([]string, len(subArr))
				for j, item := range subArr {
					switch num := item.(type) {
					case float64:
						if float64(int(num)) != num {
							return "", fmt.Errorf("non-integer number in array at index %d,%d: %v", i, j, num)
						}
						subParts[j] = fmt.Sprintf("%d", int(num))
					case int:
						subParts[j] = fmt.Sprintf("%d", num)
					case string:
						subParts[j] = num
					default:
						return "", fmt.Errorf("unsupported type in array at index %d,%d: %T", i, j, item)
					}
				}
				parts[i] = strings.Join(subParts, ",")
			}
			hashInput = strings.Join(parts, ";")
		} else {
			// 1次元配列の処理
			parts := make([]string, len(v))
			for i, item := range v {
				switch num := item.(type) {
				case float64:
					if float64(int(num)) != num {
						return "", fmt.Errorf("non-integer number at index %d: %v", i, num)
					}
					parts[i] = fmt.Sprintf("%d", int(num))
				case string:
					parts[i] = num
				case int:
					parts[i] = fmt.Sprintf("%d", num)
				default:
					return "", fmt.Errorf("unsupported type at index %d: %T", i, item)
				}
			}
			hashInput = strings.Join(parts, ",")
		}
	default:
		return "", fmt.Errorf("unsupported input type: %T", input)
	}

	// 単純なハッシュ計算（テスト用）
	return fmt.Sprintf("test-hash-%s", hashInput), nil
}

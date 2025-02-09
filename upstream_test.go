package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCreateEmbedding(t *testing.T) {
	// モックサーバーを設定
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// リクエストの検証
		if r.Method != http.MethodPost {
			t.Errorf("Expected POST request, got %s", r.Method)
		}
		if r.Header.Get("Authorization") == "" {
			t.Error("Missing Authorization header")
		}

		// リクエストボディの検証
		var req EmbeddingRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Errorf("Failed to decode request body: %v", err)
		}

		// エラーケースのテスト
		if req.Model == "error-model" {
			errResp := ErrorResponse{}
			errResp.Error.Message = "Invalid model"
			errResp.Error.Type = "invalid_request_error"
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(errResp)
			return
		}

		// 正常系のレスポンス
		resp := EmbeddingResponse{
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
			Model: req.Model,
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

	tests := []struct {
		name           string
		request        *EmbeddingRequest
		authHeader     string
		wantError      bool
		wantStatusCode int
		wantErrorType  string
	}{
		{
			name: "valid request returns embedding",
			request: &EmbeddingRequest{
				Input: "The food was delicious",
				Model: "text-embedding-ada-002",
			},
			authHeader: "Bearer sk-valid-key",
			wantError:  false,
		},
		{
			name: "upstream error is properly handled",
			request: &EmbeddingRequest{
				Input: "The food was delicious",
				Model: "error-model",
			},
			authHeader:     "Bearer sk-valid-key",
			wantError:      true,
			wantStatusCode: http.StatusBadRequest,
			wantErrorType:  "invalid_request_error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := newUpstreamClient(ts.URL)
			resp, err := client.createEmbedding(tt.request, tt.authHeader)

			if tt.wantError {
				if err == nil {
					t.Error("Expected error but got none")
					return
				}
				if ue, ok := err.(*upstreamError); ok {
					if ue.statusCode != tt.wantStatusCode {
						t.Errorf("Wrong status code: got %v want %v", ue.statusCode, tt.wantStatusCode)
					}
					if ue.response.Error.Type != tt.wantErrorType {
						t.Errorf("Wrong error type: got %v want %v", ue.response.Error.Type, tt.wantErrorType)
					}
				} else {
					t.Errorf("Expected upstreamError but got %T", err)
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

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

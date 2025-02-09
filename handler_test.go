package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHandleEmbeddings(t *testing.T) {
	// モックサーバーの設定（シンプルな成功レスポンスのみ）
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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

	dimensions := 1536
	allowedModels := []string{"text-embedding-ada-002"}
	apiKeyPattern := "^sk-[a-zA-Z0-9]{32}$"
	validAPIKey := "sk-abcdefghijklmnopqrstuvwxyz123456" // 有効なAPIキーの例

	tests := []struct {
		name          string
		method        string
		path          string
		body          *EmbeddingRequest
		authHeader    string
		wantStatus    int
		wantErrorMsg  string
		wantErrorType string
	}{
		{
			name:   "valid request returns 200",
			method: http.MethodPost,
			path:   "/v1/embeddings",
			body: &EmbeddingRequest{
				Input: "The food was delicious and the waiter...",
				Model: "text-embedding-ada-002",
			},
			authHeader: "Bearer " + validAPIKey,
			wantStatus: http.StatusOK,
		},
		{
			name:          "missing auth header returns 401",
			method:        http.MethodPost,
			path:          "/v1/embeddings",
			body:          nil,
			authHeader:    "",
			wantStatus:    http.StatusUnauthorized,
			wantErrorMsg:  "Missing or invalid Authorization header. Expected format: 'Bearer YOUR-API-KEY'",
			wantErrorType: "invalid_request_error",
		},
		{
			name:          "invalid auth format returns 401",
			method:        http.MethodPost,
			path:          "/v1/embeddings",
			body:          nil,
			authHeader:    "Basic " + apiKeyPattern,
			wantStatus:    http.StatusUnauthorized,
			wantErrorMsg:  "Missing or invalid Authorization header. Expected format: 'Bearer YOUR-API-KEY'",
			wantErrorType: "invalid_request_error",
		},
		{
			name:          "invalid api key returns 401",
			method:        http.MethodPost,
			path:          "/v1/embeddings",
			body:          nil,
			authHeader:    "Bearer invalid-key",
			wantStatus:    http.StatusUnauthorized,
			wantErrorMsg:  "Invalid API key",
			wantErrorType: "invalid_request_error",
		},
		{
			name:   "valid request with optional params returns 200",
			method: http.MethodPost,
			path:   "/v1/embeddings",
			body: &EmbeddingRequest{
				Input:          "The food was delicious and the waiter...",
				Model:          "text-embedding-ada-002",
				EncodingFormat: "float",
				Dimensions:     &dimensions,
				User:           "user-123",
			},
			authHeader: "Bearer " + validAPIKey,
			wantStatus: http.StatusOK,
		},
		{
			name:   "invalid encoding_format returns 400",
			method: http.MethodPost,
			path:   "/v1/embeddings",
			body: &EmbeddingRequest{
				Input:          "The food was delicious",
				Model:          "text-embedding-ada-002",
				EncodingFormat: "invalid",
			},
			authHeader:    "Bearer " + validAPIKey,
			wantStatus:    http.StatusBadRequest,
			wantErrorMsg:  "Invalid encoding_format: must be either 'float' or 'base64'",
			wantErrorType: "invalid_request_error",
		},
		{
			name:   "empty input returns 400",
			method: http.MethodPost,
			path:   "/v1/embeddings",
			body: &EmbeddingRequest{
				Input: "",
				Model: "text-embedding-ada-002",
			},
			authHeader:    "Bearer " + validAPIKey,
			wantStatus:    http.StatusBadRequest,
			wantErrorMsg:  "Missing required fields: 'input' and 'model' must not be empty",
			wantErrorType: "invalid_request_error",
		},
		{
			name:   "empty model returns 400",
			method: http.MethodPost,
			path:   "/v1/embeddings",
			body: &EmbeddingRequest{
				Input: "The food was delicious",
				Model: "",
			},
			authHeader:    "Bearer " + validAPIKey,
			wantStatus:    http.StatusBadRequest,
			wantErrorMsg:  "Missing required fields: 'input' and 'model' must not be empty",
			wantErrorType: "invalid_request_error",
		},
		{
			name:          "invalid JSON returns 400",
			method:        http.MethodPost,
			path:          "/v1/embeddings",
			body:          nil,
			authHeader:    "Bearer " + validAPIKey,
			wantStatus:    http.StatusBadRequest,
			wantErrorMsg:  "Invalid JSON payload",
			wantErrorType: "invalid_request_error",
		},
		{
			name:       "GET to correct path returns 405",
			method:     http.MethodGet,
			path:       "/v1/embeddings",
			body:       nil,
			wantStatus: http.StatusMethodNotAllowed,
		},
		{
			name:       "wrong path returns 404",
			method:     http.MethodPost,
			path:       "/wrong/path",
			body:       nil,
			wantStatus: http.StatusNotFound,
		},
		{
			name:   "unsupported model returns 400",
			method: http.MethodPost,
			path:   "/v1/embeddings",
			body: &EmbeddingRequest{
				Input: "The food was delicious",
				Model: "unsupported-model",
			},
			authHeader:    "Bearer " + validAPIKey,
			wantStatus:    http.StatusBadRequest,
			wantErrorMsg:  "Unsupported model: unsupported-model",
			wantErrorType: "invalid_request_error",
		},
		{
			name:          "invalid api key format returns 401",
			method:        http.MethodPost,
			path:          "/v1/embeddings",
			body:          nil,
			authHeader:    "Bearer invalid-format-key",
			wantStatus:    http.StatusUnauthorized,
			wantErrorMsg:  "Invalid API key format",
			wantErrorType: "invalid_request_error",
		},
		{
			name:   "valid api key format returns 200",
			method: http.MethodPost,
			path:   "/v1/embeddings",
			body: &EmbeddingRequest{
				Input: "The food was delicious",
				Model: "text-embedding-ada-002",
			},
			authHeader: "Bearer " + validAPIKey,
			wantStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var body []byte
			var err error
			if tt.body != nil {
				body, err = json.Marshal(tt.body)
				if err != nil {
					t.Fatal(err)
				}
			}

			req, err := http.NewRequest(tt.method, tt.path, bytes.NewBuffer(body))
			if err != nil {
				t.Fatal(err)
			}

			if tt.body != nil {
				req.Header.Set("Content-Type", "application/json")
			}
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}

			rr := httptest.NewRecorder()
			handler := newHandler(allowedModels, apiKeyPattern, ts.URL)
			handler.ServeHTTP(rr, req)

			if status := rr.Code; status != tt.wantStatus {
				t.Errorf("handler returned wrong status code: got %v want %v",
					status, tt.wantStatus)
			}

			if tt.wantErrorMsg != "" {
				var errResp ErrorResponse
				if err := json.NewDecoder(rr.Body).Decode(&errResp); err != nil {
					t.Fatalf("Failed to decode error response: %v", err)
				}

				if !strings.Contains(errResp.Error.Message, tt.wantErrorMsg) {
					t.Errorf("handler returned wrong error message: got %v want %v",
						errResp.Error.Message, tt.wantErrorMsg)
				}

				if errResp.Error.Type != tt.wantErrorType {
					t.Errorf("handler returned wrong error type: got %v want %v",
						errResp.Error.Type, tt.wantErrorType)
				}
			}
		})
	}
}

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
	dimensions := 1536
	allowedModels := []string{"text-embedding-ada-002"}

	tests := []struct {
		name          string
		method        string
		path          string
		body          *EmbeddingRequest
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
			wantStatus: http.StatusOK,
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
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "invalid JSON returns 400",
			method:     http.MethodPost,
			path:       "/v1/embeddings",
			body:       nil,
			wantStatus: http.StatusBadRequest,
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
			wantStatus:    http.StatusBadRequest,
			wantErrorMsg:  "Unsupported model: unsupported-model",
			wantErrorType: "invalid_request_error",
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

			rr := httptest.NewRecorder()
			handler := newHandler(allowedModels)
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

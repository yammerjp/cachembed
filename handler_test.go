package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHandleEmbeddings(t *testing.T) {
	tests := []struct {
		name       string
		method     string
		path       string
		wantStatus int
	}{
		{
			name:       "POST to correct path returns 200",
			method:     http.MethodPost,
			path:       "/v1/embeddings",
			wantStatus: http.StatusOK,
		},
		{
			name:       "GET to correct path returns 405",
			method:     http.MethodGet,
			path:       "/v1/embeddings",
			wantStatus: http.StatusMethodNotAllowed,
		},
		{
			name:       "wrong path returns 404",
			method:     http.MethodPost,
			path:       "/wrong/path",
			wantStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest(tt.method, tt.path, nil)
			if err != nil {
				t.Fatal(err)
			}

			rr := httptest.NewRecorder()
			handler := http.HandlerFunc(handleEmbeddings)
			handler.ServeHTTP(rr, req)

			if status := rr.Code; status != tt.wantStatus {
				t.Errorf("handler returned wrong status code: got %v want %v",
					status, tt.wantStatus)
			}
		})
	}
}

package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHandleEmbeddings(t *testing.T) {
	// Create a request to pass to our handler
	req, err := http.NewRequest("POST", "/v1/embeddings", nil)
	if err != nil {
		t.Fatal(err)
	}

	// Create a ResponseRecorder to record the response
	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(handleEmbeddings)

	// Call the handler directly with the request and response recorder
	handler.ServeHTTP(rr, req)

	// Check the status code
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}
}

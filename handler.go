package main

import (
	"encoding/json"
	"net/http"
	"slices"
)

type EmbeddingRequest struct {
	Input          string `json:"input"`
	Model          string `json:"model"`
	EncodingFormat string `json:"encoding_format,omitempty"`
	Dimensions     *int   `json:"dimensions,omitempty"`
	User           string `json:"user,omitempty"`
}

type ErrorResponse struct {
	Error struct {
		Message string `json:"message"`
		Type    string `json:"type"`
		Code    string `json:"code"`
	} `json:"error"`
}

type handler struct {
	allowedModels []string
}

func newHandler(allowedModels []string) http.Handler {
	return &handler{
		allowedModels: allowedModels,
	}
}

func writeError(w http.ResponseWriter, status int, message, errType string) {
	var resp ErrorResponse
	resp.Error.Message = message
	resp.Error.Type = errType
	resp.Error.Code = http.StatusText(status)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(resp)
}

func (h *handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/v1/embeddings" {
		writeError(w, http.StatusNotFound, "Not found", "invalid_request_error")
		return
	}

	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed. Please use POST.", "invalid_request_error")
		return
	}

	var req EmbeddingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON payload: "+err.Error(), "invalid_request_error")
		return
	}

	if req.Input == "" || req.Model == "" {
		writeError(w, http.StatusBadRequest, "Missing required fields: 'input' and 'model' must not be empty", "invalid_request_error")
		return
	}

	if !slices.Contains(h.allowedModels, req.Model) {
		writeError(w, http.StatusBadRequest, "Unsupported model: "+req.Model, "invalid_request_error")
		return
	}

	if req.EncodingFormat != "" && req.EncodingFormat != "float" && req.EncodingFormat != "base64" {
		writeError(w, http.StatusBadRequest, "Invalid encoding_format: must be either 'float' or 'base64'", "invalid_request_error")
		return
	}

	w.WriteHeader(http.StatusOK)
}

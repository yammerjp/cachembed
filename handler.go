package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"regexp"
	"slices"
	"strings"

	"github.com/google/uuid"
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

type EmbeddingResponse struct {
	Object string `json:"object"`
	Data   []struct {
		Object    string    `json:"object"`
		Embedding []float32 `json:"embedding"`
		Index     int       `json:"index"`
	} `json:"data"`
	Model string `json:"model"`
	Usage struct {
		PromptTokens int `json:"prompt_tokens"`
		TotalTokens  int `json:"total_tokens"`
	} `json:"usage"`
}

type handler struct {
	allowedModels []string
	apiKeyPattern string
	apiKeyRegexp  *regexp.Regexp
	upstream      *upstreamClient
}

func newHandler(allowedModels []string, apiKeyPattern string, upstreamURL string) http.Handler {
	var re *regexp.Regexp
	if apiKeyPattern != "" {
		var err error
		re, err = regexp.Compile(apiKeyPattern)
		if err != nil {
			log.Fatalf("Invalid API key pattern: %v", err)
			os.Exit(1)
		}
	}
	return &handler{
		allowedModels: allowedModels,
		apiKeyPattern: apiKeyPattern,
		apiKeyRegexp:  re,
		upstream:      newUpstreamClient(upstreamURL),
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
	requestID := uuid.New().String()
	ctx := r.Context()
	ctx = context.WithValue(ctx, "request_id", requestID)
	r = r.WithContext(ctx)

	result := &requestResult{
		path:   r.URL.Path,
		method: r.Method,
	}
	defer func() {
		// リクエスト完了時に1つのログエントリを出力
		logger := slog.With(
			"request_id", requestID,
			"path", result.path,
			"method", result.method,
			"status", result.status,
		)

		// 成功時のみトークン使用量を記録
		attrs := []any{}
		if result.status == http.StatusOK {
			attrs = append(attrs,
				"prompt_tokens", result.promptTokens,
				"total_tokens", result.totalTokens,
			)
		}

		// 5xx系エラーの場合のみエラー詳細を記録
		if result.status >= 500 && result.err != nil {
			attrs = append(attrs, "error", result.err.Error())
		}

		// 5xx系エラーの場合はエラーレベル、それ以外は情報レベル
		if result.status >= 500 {
			logger.Error("request completed", attrs...)
		} else {
			logger.Info("request completed", attrs...)
		}
	}()

	if err := h.handleRequest(w, r, result); err != nil && result.status >= 500 {
		slog.Debug("request processing error",
			"request_id", requestID,
			"error", err,
		)
	}
}

type requestResult struct {
	path         string
	method       string
	status       int
	err          error
	promptTokens int
	totalTokens  int
}

func (h *handler) handleRequest(w http.ResponseWriter, r *http.Request, result *requestResult) error {
	if r.URL.Path != "/v1/embeddings" {
		result.status = http.StatusNotFound
		result.err = fmt.Errorf("not found")
		writeError(w, result.status, "Not found", "invalid_request_error")
		return result.err
	}

	if r.Method != http.MethodPost {
		result.status = http.StatusMethodNotAllowed
		result.err = fmt.Errorf("method not allowed: %s", r.Method)
		writeError(w, result.status, "Method not allowed. Please use POST.", "invalid_request_error")
		return result.err
	}

	// Check Authorization header
	authHeader := r.Header.Get("Authorization")
	if !strings.HasPrefix(authHeader, "Bearer ") {
		result.status = http.StatusUnauthorized
		result.err = fmt.Errorf("invalid auth header format")
		writeError(w, result.status, "Missing or invalid Authorization header. Expected format: 'Bearer YOUR-API-KEY'", "invalid_request_error")
		return result.err
	}

	token := strings.TrimPrefix(authHeader, "Bearer ")
	if token == "" {
		result.status = http.StatusUnauthorized
		result.err = fmt.Errorf("empty api key")
		writeError(w, result.status, "API key is required", "invalid_request_error")
		return result.err
	}

	if h.apiKeyRegexp != nil && !h.apiKeyRegexp.MatchString(token) {
		result.status = http.StatusUnauthorized
		result.err = fmt.Errorf("invalid api key format")
		writeError(w, result.status, "Invalid API key format", "invalid_request_error")
		return result.err
	}

	var req EmbeddingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		result.status = http.StatusBadRequest
		result.err = fmt.Errorf("invalid json: %w", err)
		writeError(w, result.status, "Invalid JSON payload: "+err.Error(), "invalid_request_error")
		return result.err
	}

	if req.Input == "" || req.Model == "" {
		result.status = http.StatusBadRequest
		result.err = fmt.Errorf("missing required fields")
		writeError(w, result.status, "Missing required fields: 'input' and 'model' must not be empty", "invalid_request_error")
		return result.err
	}

	if !slices.Contains(h.allowedModels, req.Model) {
		result.status = http.StatusBadRequest
		result.err = fmt.Errorf("unsupported model: %s", req.Model)
		writeError(w, result.status, "Unsupported model: "+req.Model, "invalid_request_error")
		return result.err
	}

	if req.EncodingFormat != "" && req.EncodingFormat != "float" && req.EncodingFormat != "base64" {
		result.status = http.StatusBadRequest
		result.err = fmt.Errorf("invalid encoding format: %s", req.EncodingFormat)
		writeError(w, result.status, "Invalid encoding_format: must be either 'float' or 'base64'", "invalid_request_error")
		return result.err
	}

	// OpenAI APIにリクエストを送信
	resp, err := h.upstream.createEmbedding(&req, r.Header.Get("Authorization"))
	if err != nil {
		if ue, ok := err.(*upstreamError); ok {
			result.status = ue.statusCode
			result.err = fmt.Errorf("upstream error: %w", err)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(ue.statusCode)
			json.NewEncoder(w).Encode(ue.response)
			return result.err
		}
		result.status = http.StatusBadGateway
		result.err = fmt.Errorf("upstream error: %w", err)
		writeError(w, result.status, "Failed to reach upstream API: "+err.Error(), "upstream_error")
		return result.err
	}

	// 成功時のメタデータを記録
	result.status = http.StatusOK
	result.promptTokens = resp.Usage.PromptTokens
	result.totalTokens = resp.Usage.TotalTokens

	// レスポンスを返す
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	return json.NewEncoder(w).Encode(resp)
}

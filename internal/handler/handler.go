package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"log/slog"
	"net/http"
	"os"
	"regexp"
	"slices"
	"strings"

	"crypto/sha1"
	"encoding/hex"

	"github.com/google/uuid"
	"github.com/yammerjp/cachembed/internal/storage"
	"github.com/yammerjp/cachembed/internal/upstream"
)

type Handler struct {
	allowedModels []string
	apiKeyPattern string
	apiKeyRegexp  *regexp.Regexp
	upstream      *upstream.Client
	db            *storage.DB
	debugBody     bool
}

func NewHandler(allowedModels []string, apiKeyPattern string, upstreamURL string, db *storage.DB, debugBody bool) http.Handler {
	var re *regexp.Regexp
	if apiKeyPattern != "" {
		var err error
		re, err = regexp.Compile(apiKeyPattern)
		if err != nil {
			log.Fatalf("Invalid API key pattern: %v", err)
			os.Exit(1)
		}
	}
	return &Handler{
		allowedModels: allowedModels,
		apiKeyPattern: apiKeyPattern,
		apiKeyRegexp:  re,
		upstream:      upstream.NewClient(upstreamURL),
		db:            db,
		debugBody:     debugBody,
	}
}

func writeError(w http.ResponseWriter, status int, message, errType string) {
	var resp upstream.ErrorResponse
	resp.Error.Message = message
	resp.Error.Type = errType
	resp.Error.Code = http.StatusText(status)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(resp)
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
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

func (h *Handler) handleRequest(w http.ResponseWriter, r *http.Request, result *requestResult) error {
	// debug payload
	body, err := io.ReadAll(r.Body)
	if err != nil {
		result.status = http.StatusBadRequest
		result.err = fmt.Errorf("failed to read request body: %w", err)
		writeError(w, result.status, "Failed to read request body", "invalid_request_error")
		return result.err
	}
	if h.debugBody {
		slog.Debug("request payload", "payload", string(body))
	}

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

	var req upstream.EmbeddingRequest
	if err := json.NewDecoder(bytes.NewReader(body)).Decode(&req); err != nil {
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

	// 入力のハッシュを計算
	inputHash := sha1.Sum([]byte(req.Input))
	inputHashStr := hex.EncodeToString(inputHash[:])

	// キャッシュをチェック
	if cache, err := h.db.GetEmbedding(inputHashStr, req.Model); err != nil {
		slog.Error("failed to query cache",
			"error", err,
			"input_hash", inputHashStr,
			"model", req.Model,
		)
	} else if cache != nil {
		// キャッシュヒット
		slog.Debug("cache hit",
			"input_hash", inputHashStr,
			"model", req.Model,
			"created_at", cache.CreatedAt,
			"last_accessed", cache.LastAccessed,
		)

		resp := upstream.EmbeddingResponse{
			Object: "list",
			Data: []struct {
				Object    string    `json:"object"`
				Embedding []float32 `json:"embedding"`
				Index     int       `json:"index"`
			}{
				{
					Object:    "embedding",
					Embedding: cache.EmbeddingData,
					Index:     0,
				},
			},
			Model: req.Model,
			Usage: struct {
				PromptTokens int `json:"prompt_tokens"`
				TotalTokens  int `json:"total_tokens"`
			}{
				// キャッシュヒット時はトークン数を0として報告
				PromptTokens: 0,
				TotalTokens:  0,
			},
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		return json.NewEncoder(w).Encode(resp)
	}

	// キャッシュミス：upstreamにリクエスト
	resp, err := h.upstream.CreateEmbedding(&req, r.Header.Get("Authorization"))
	if err != nil {
		if ue, ok := err.(*upstream.UpstreamError); ok {
			result.status = ue.StatusCode
			result.err = fmt.Errorf("upstream error: %w", err)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(ue.StatusCode)
			json.NewEncoder(w).Encode(ue.Response)
			return result.err
		}
		result.status = http.StatusBadGateway
		result.err = fmt.Errorf("upstream error: %w", err)
		writeError(w, result.status, "Failed to reach upstream API: "+err.Error(), "upstream_error")
		return result.err
	}

	// 成功時はキャッシュに保存
	if err := h.db.StoreEmbedding(inputHashStr, req.Model, resp.Data[0].Embedding); err != nil {
		slog.Error("failed to store cache",
			"error", err,
			"input_hash", inputHashStr,
			"model", req.Model,
		)
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

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

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	requestID := uuid.New().String()
	ctx := r.Context()
	ctx = context.WithValue(ctx, "request_id", requestID)
	r = r.WithContext(ctx)

	err := h.handleRequest(w, r)

	// ログ出力用の属性を準備
	attrs := []any{
		"request_id", requestID,
		"path", r.URL.Path,
		"method", r.Method,
	}

	if err == nil {
		// 成功時
		attrs = append(attrs, "status", http.StatusOK)
		slog.Info("request completed", attrs...)
		return
	}

	// エラー時
	if handlerErr, ok := err.(*HandlerError); ok {
		attrs = append(attrs, "status", handlerErr.Status)

		// トークン使用量があれば記録
		if handlerErr.PromptTokens > 0 {
			attrs = append(attrs,
				"prompt_tokens", handlerErr.PromptTokens,
				"total_tokens", handlerErr.TotalTokens,
			)
		}

		// 5xx系エラーの場合のみ詳細なエラー情報を記録
		if handlerErr.Status >= 500 {
			attrs = append(attrs, "error", handlerErr.Error())
			slog.Error("request completed", attrs...)
		} else {
			slog.Info("request completed", attrs...)
		}

		handlerErr.WriteResponse(w)
	} else {
		// 予期せぬエラー
		attrs = append(attrs,
			"status", http.StatusInternalServerError,
			"error", err.Error(),
		)
		slog.Error("unexpected error", attrs...)
		writeError(w, http.StatusInternalServerError, "Internal server error", "internal_error")
	}
}

func (h *Handler) handleRequest(w http.ResponseWriter, r *http.Request) error {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return NewHandlerError(
			http.StatusBadRequest,
			"Failed to read request body",
			"invalid_request_error",
			err,
		)
	}
	if h.debugBody {
		slog.Debug("request payload", "payload", string(body))
	}

	if err := h.validateRequest(r); err != nil {
		return err
	}

	var req upstream.EmbeddingRequest
	if err := json.NewDecoder(bytes.NewReader(body)).Decode(&req); err != nil {
		return NewHandlerError(
			http.StatusBadRequest,
			"Invalid JSON payload: "+err.Error(),
			"invalid_request_error",
			err,
		)
	}

	if err := h.validateEmbeddingRequest(&req); err != nil {
		return err
	}

	// 入力をハッシュ化してキャッシュを確認
	inputs, err := processInput(req.Input)
	if err != nil {
		return NewHandlerError(
			http.StatusBadRequest,
			"Invalid input: "+err.Error(),
			"invalid_request_error",
			err,
		)
	}

	// レスポンスの準備
	resp := &upstream.EmbeddingResponse{
		Object: "list",
		Data:   make([]upstream.EmbeddingData, len(inputs)),
		Model:  req.Model,
		Usage:  upstream.Usage{},
	}

	// キャッシュミスした入力を収集
	var missedInputs []string
	var missedIndices []int

	// 各入力に対してキャッシュを確認
	for i, input := range inputs {
		inputHash := sha1.Sum([]byte(input))
		inputHashStr := hex.EncodeToString(inputHash[:])

		cache, err := h.db.GetEmbedding(inputHashStr, req.Model)
		if err != nil {
			return NewHandlerError(
				http.StatusInternalServerError,
				"Failed to check cache",
				"internal_error",
				err,
			)
		}

		if cache != nil {
			// キャッシュヒットの場合
			float64Embedding := make([]float64, len(cache.EmbeddingData))
			for j, v := range cache.EmbeddingData {
				float64Embedding[j] = float64(v)
			}
			resp.Data[i] = upstream.EmbeddingData{
				Object:    "embedding",
				Embedding: float64Embedding,
				Index:     i,
			}
		} else {
			// キャッシュミスの場合
			missedInputs = append(missedInputs, input)
			missedIndices = append(missedIndices, i)
		}
	}

	// キャッシュミスがある場合、APIリクエストを実行
	if len(missedInputs) > 0 {
		missedReq := upstream.EmbeddingRequest{
			Input:          missedInputs,
			Model:          req.Model,
			EncodingFormat: req.EncodingFormat,
		}
		missedResp, err := h.upstream.CreateEmbedding(&missedReq, r.Header.Get("Authorization"))
		if err != nil {
			if ue, ok := err.(*upstream.UpstreamError); ok {
				return NewHandlerErrorWithTokens(
					ue.StatusCode,
					"Failed to reach upstream API: "+err.Error(),
					"upstream_error",
					err,
					ue.Usage.PromptTokens,
					ue.Usage.TotalTokens,
				)
			}
			return NewHandlerError(
				http.StatusBadGateway,
				"Failed to reach upstream API: "+err.Error(),
				"upstream_error",
				err,
			)
		}

		// APIレスポンスをキャッシュに保存し、レスポンスに組み込む
		for i, data := range missedResp.Data {
			embedding, ok := convertToFloat32Slice(data.Embedding)
			if !ok {
				return NewHandlerError(
					http.StatusInternalServerError,
					"Unexpected embedding type from upstream",
					"internal_error",
					fmt.Errorf("unexpected embedding type from upstream at index %d", i),
				)
			}

			// キャッシュに保存
			inputHash := sha1.Sum([]byte(missedInputs[i]))
			inputHashStr := hex.EncodeToString(inputHash[:])
			if err := h.db.StoreEmbedding(inputHashStr, req.Model, embedding); err != nil {
				slog.Error("failed to store cache",
					"error", err,
					"input_hash", inputHashStr,
					"model", req.Model,
					"index", i,
				)
			}

			// レスポンスに組み込む
			originalIndex := missedIndices[i]
			resp.Data[originalIndex] = upstream.EmbeddingData{
				Object:    "embedding",
				Embedding: data.Embedding,
				Index:     originalIndex,
			}
		}

		// 使用量を記録
		resp.Usage = missedResp.Usage
	}

	// base64エンコーディングが要求された場合の処理
	if req.EncodingFormat == "base64" {
		for i, data := range resp.Data {
			embedding, ok := convertToFloat32Slice(data.Embedding)
			if ok {
				resp.Data[i].Embedding = float32ToBase64(embedding)
			}
		}
	}

	// レスポンスを返す
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	return json.NewEncoder(w).Encode(resp)
}

func (h *Handler) validateRequest(r *http.Request) error {
	if r.URL.Path != "/v1/embeddings" {
		return NewHandlerError(
			http.StatusNotFound,
			"Not found",
			"invalid_request_error",
			fmt.Errorf("not found"),
		)
	}

	if r.Method != http.MethodPost {
		return NewHandlerError(
			http.StatusMethodNotAllowed,
			"Method not allowed. Please use POST.",
			"invalid_request_error",
			fmt.Errorf("method not allowed: %s", r.Method),
		)
	}

	// Authorization headerのチェック
	authHeader := r.Header.Get("Authorization")
	if !strings.HasPrefix(authHeader, "Bearer ") {
		return NewHandlerError(
			http.StatusUnauthorized,
			"Missing or invalid Authorization header. Expected format: 'Bearer YOUR-API-KEY'",
			"invalid_request_error",
			fmt.Errorf("invalid auth header format"),
		)
	}

	token := strings.TrimPrefix(authHeader, "Bearer ")
	if token == "" {
		return NewHandlerError(
			http.StatusUnauthorized,
			"API key is required",
			"invalid_request_error",
			fmt.Errorf("empty api key"),
		)
	}

	if h.apiKeyRegexp != nil && !h.apiKeyRegexp.MatchString(token) {
		return NewHandlerError(
			http.StatusUnauthorized,
			"Invalid API key format",
			"invalid_request_error",
			fmt.Errorf("invalid api key format"),
		)
	}

	return nil
}

func (h *Handler) validateEmbeddingRequest(req *upstream.EmbeddingRequest) error {
	if req.Input == nil || req.Model == "" {
		return NewHandlerError(
			http.StatusBadRequest,
			"Missing required fields: 'input' and 'model' must not be empty",
			"invalid_request_error",
			fmt.Errorf("missing required fields"),
		)
	}

	if !slices.Contains(h.allowedModels, req.Model) {
		return NewHandlerError(
			http.StatusBadRequest,
			"Unsupported model: "+req.Model,
			"invalid_request_error",
			fmt.Errorf("unsupported model: %s", req.Model),
		)
	}

	if req.EncodingFormat != "" && req.EncodingFormat != "float" && req.EncodingFormat != "base64" {
		return NewHandlerError(
			http.StatusBadRequest,
			"Invalid encoding_format: must be either 'float' or 'base64'",
			"invalid_request_error",
			fmt.Errorf("invalid encoding format: %s", req.EncodingFormat),
		)
	}

	return nil
}

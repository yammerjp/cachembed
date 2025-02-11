package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"regexp"
	"slices"
	"strings"

	"github.com/google/uuid"
	"github.com/yammerjp/cachembed/internal/storage"
	"github.com/yammerjp/cachembed/internal/upstream"
)

type Handler struct {
	allowedModels []string
	apiKeyRegexp  *regexp.Regexp
	upstream      upstream.EmbeddingClient
	db            storage.Database
	debugBody     bool
}

func NewHandler(allowedModels []string, apiKeyRegexp *regexp.Regexp, db storage.Database, upstream upstream.EmbeddingClient, debugBody bool) *Handler {
	return &Handler{
		allowedModels: allowedModels,
		apiKeyRegexp:  apiKeyRegexp,
		upstream:      upstream,
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

	// JSONデコード時に数値を適切に処理するために、一度rawMessageとして受け取る
	var rawReq struct {
		Input          json.RawMessage `json:"input"`
		Model          string          `json:"model"`
		EncodingFormat string          `json:"encoding_format,omitempty"`
	}
	if err := json.NewDecoder(bytes.NewReader(body)).Decode(&rawReq); err != nil {
		return NewHandlerError(
			http.StatusBadRequest,
			"Invalid JSON payload: "+err.Error(),
			"invalid_request_error",
			err,
		)
	}

	// 入力値をパースして適切な型に変換
	input, err := ParseEmbeddingInput(rawReq.Input)
	if err != nil {
		return NewHandlerError(
			http.StatusBadRequest,
			"Invalid input format: "+err.Error(),
			"invalid_request_error",
			err,
		)
	}

	// リクエストオブジェクトを構築
	req := upstream.EmbeddingRequest{
		Input:          input.ToAPIInput(),
		Model:          rawReq.Model,
		EncodingFormat: "base64",
	}

	if err := h.validateEmbeddingRequest(&req); err != nil {
		return err
	}

	hashes, err := input.InputHashes()
	if err != nil {
		return NewHandlerError(
			http.StatusBadRequest,
			"Invalid input: "+err.Error(),
			"invalid_request_error",
			err,
		)
	}

	responseEmbeddingDatas := make([]upstream.EmbeddingData, len(hashes))
	missedIndexes := make([]int, 0)
	usage := upstream.Usage{}
	for i, hash := range hashes {
		cache, err := h.db.GetEmbedding(hash, req.Model)
		if err != nil {
			return NewHandlerError(
				http.StatusInternalServerError,
				"Failed to check cache",
				"internal_error",
				err,
			)
		}
		if cache != nil {
			var responseEmbedding interface{}
			if rawReq.EncodingFormat == "base64" {
				responseEmbedding = float32ToBase64(cache)
			} else {
				responseEmbedding = cache
			}
			responseEmbeddingDatas[i] = upstream.EmbeddingData{
				Object:    "embedding",
				Embedding: responseEmbedding,
				Index:     i,
			}
		} else {
			missedIndexes = append(missedIndexes, i)
		}
	}

	// キャッシュミスがある場合、APIリクエストを実行
	if len(missedIndexes) > 0 {
		pickedInput, err := input.PickInputs(missedIndexes)
		if err != nil {
			return NewHandlerError(
				http.StatusInternalServerError,
				"Failed to pick inputs",
				"internal_error",
				err,
			)
		}
		missedReq := upstream.EmbeddingRequest{
			Input:          pickedInput.ToAPIInput(),
			Model:          req.Model,
			EncodingFormat: "base64",
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
			// base64からfloat32に変換
			var embedding []float32
			switch v := data.Embedding.(type) {
			case string:
				// base64エンコードされた文字列の場合
				var err error
				embedding, err = base64ToFloat32Slice(v)
				if err != nil {
					return NewHandlerError(
						http.StatusInternalServerError,
						"Failed to decode base64 embedding from upstream",
						"internal_error",
						err,
					)
				}
			case []interface{}:
				// float64の配列の場合
				embedding = make([]float32, len(v))
				for j, val := range v {
					switch num := val.(type) {
					case float64:
						embedding[j] = float32(num)
					default:
						return NewHandlerError(
							http.StatusInternalServerError,
							"Invalid embedding type from upstream",
							"internal_error",
							fmt.Errorf("unexpected type in embedding array: %T", val),
						)
					}
				}
			default:
				return NewHandlerError(
					http.StatusInternalServerError,
					"Invalid embedding type from upstream",
					"internal_error",
					fmt.Errorf("unexpected embedding type: %T", data.Embedding),
				)
			}

			// キャッシュに保存
			if err := h.db.StoreEmbedding(hashes[missedIndexes[i]], req.Model, embedding); err != nil {
				slog.Error("failed to store cache",
					"error", err,
					"input_hash", hashes[missedIndexes[i]],
					"model", req.Model,
					"index", i,
				)
			}

			// クライアントのリクエストに応じたフォーマットでレスポンスを設定
			var responseEmbedding interface{}
			if rawReq.EncodingFormat == "base64" {
				responseEmbedding = data.Embedding
			} else {
				responseEmbedding = embedding
			}

			// レスポンスに組み込む
			responseEmbeddingDatas[missedIndexes[i]] = upstream.EmbeddingData{
				Object:    "embedding",
				Embedding: responseEmbedding,
				Index:     missedIndexes[i],
			}
		}

		// 使用量を記録
		usage = missedResp.Usage
	}

	var response upstream.EmbeddingResponse
	if len(hashes) == 1 {
		response.Data = []upstream.EmbeddingData{
			{
				Object:    "embedding",
				Embedding: responseEmbeddingDatas[0].Embedding,
			},
		}
	} else {
		response.Data = responseEmbeddingDatas
	}
	response.Usage = usage

	// レスポンスを返す
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	return json.NewEncoder(w).Encode(response)
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

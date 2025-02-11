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

	// 入力値を適切な型に変換
	var input interface{}
	if err := json.Unmarshal(rawReq.Input, &input); err != nil {
		return NewHandlerError(
			http.StatusBadRequest,
			"Invalid input format: "+err.Error(),
			"invalid_request_error",
			err,
		)
	}

	// 入力値の型を検証し、適切な形式に変換する
	switch v := input.(type) {
	case string:
		// 文字列の場合はそのまま
		if v == "" {
			return NewHandlerError(
				http.StatusBadRequest,
				"Input string must not be empty",
				"invalid_request_error",
				fmt.Errorf("empty string input"),
			)
		}
		input = v

	case []interface{}:
		// 配列の場合、要素の型を確認
		if len(v) == 0 {
			return NewHandlerError(
				http.StatusBadRequest,
				"Input array must not be empty",
				"invalid_request_error",
				fmt.Errorf("empty array input"),
			)
		}

		// 最初の要素の型で配列の種類を判断
		switch v[0].(type) {
		case float64, int: // 数値配列の場合
			// 全要素が数値であることを確認
			numbers := make([]float64, len(v))
			for i, item := range v {
				switch num := item.(type) {
				case float64:
					numbers[i] = num
				case int:
					numbers[i] = float64(num)
				default:
					return NewHandlerError(
						http.StatusBadRequest,
						"All elements in number array must be numbers",
						"invalid_request_error",
						fmt.Errorf("invalid element type in number array at index %d: got %T", i, item),
					)
				}
			}
			input = numbers

		case string: // 文字列配列の場合
			// 全要素が文字列であることを確認
			strings := make([]string, len(v))
			for i, item := range v {
				str, ok := item.(string)
				if !ok {
					return NewHandlerError(
						http.StatusBadRequest,
						"All elements in string array must be strings",
						"invalid_request_error",
						fmt.Errorf("invalid element type in string array at index %d: got %T", i, item),
					)
				}
				if str == "" {
					return NewHandlerError(
						http.StatusBadRequest,
						"String elements must not be empty",
						"invalid_request_error",
						fmt.Errorf("empty string at index %d", i),
					)
				}
				strings[i] = str
			}
			input = strings

		case []interface{}: // 2次元数値配列の場合
			// 全要素が数値配列であることを確認
			arrays := make([][]float64, len(v))
			for i, arr := range v {
				subArr, ok := arr.([]interface{})
				if !ok {
					return NewHandlerError(
						http.StatusBadRequest,
						"All elements must be number arrays",
						"invalid_request_error",
						fmt.Errorf("invalid element type at index %d: got %T", i, arr),
					)
				}
				numbers := make([]float64, len(subArr))
				for j, item := range subArr {
					switch num := item.(type) {
					case float64:
						numbers[j] = num
					case int:
						numbers[j] = float64(num)
					default:
						return NewHandlerError(
							http.StatusBadRequest,
							"All elements in nested arrays must be numbers",
							"invalid_request_error",
							fmt.Errorf("invalid element type at index [%d][%d]: got %T", i, j, item),
						)
					}
				}
				arrays[i] = numbers
			}
			input = arrays

		default:
			return NewHandlerError(
				http.StatusBadRequest,
				"Input array elements must be either all numbers, all strings, or all number arrays",
				"invalid_request_error",
				fmt.Errorf("invalid array element type: got %T", v[0]),
			)
		}

	default:
		return NewHandlerError(
			http.StatusBadRequest,
			"Input must be a string, an array of numbers, an array of strings, or an array of number arrays",
			"invalid_request_error",
			fmt.Errorf("invalid input type: got %T", input),
		)
	}

	// リクエストオブジェクトを構築
	req := upstream.EmbeddingRequest{
		Input:          input,
		Model:          rawReq.Model,
		EncodingFormat: rawReq.EncodingFormat,
	}

	if err := h.validateEmbeddingRequest(&req); err != nil {
		return err
	}

	hashes, err := req.InputHashes()
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
			responseEmbeddingDatas[i] = upstream.EmbeddingData{
				Object:    "embedding",
				Embedding: cache.EmbeddingData,
				Index:     i,
			}
		} else {
			missedIndexes = append(missedIndexes, i)
		}
	}

	// キャッシュミスがある場合、APIリクエストを実行
	if len(missedIndexes) > 0 {
		input, err := req.PickInputs(missedIndexes)
		if err != nil {
			return NewHandlerError(
				http.StatusInternalServerError,
				"Failed to pick inputs",
				"internal_error",
				err,
			)
		}
		missedReq := upstream.EmbeddingRequest{
			Input:          input,
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

			if err := h.db.StoreEmbedding(hashes[missedIndexes[i]], req.Model, embedding); err != nil {
				slog.Error("failed to store cache",
					"error", err,
					"input_hash", hashes[missedIndexes[i]],
					"model", req.Model,
					"index", i,
				)
			}

			// レスポンスに組み込む
			responseEmbeddingDatas[missedIndexes[i]] = data
		}

		// 使用量を記録
		usage = missedResp.Usage
	}

	// base64エンコーディングが要求された場合の処理
	if req.EncodingFormat == "base64" {
		for i, data := range responseEmbeddingDatas {
			embedding, ok := convertToFloat32Slice(data)
			if ok {
				responseEmbeddingDatas[i].Embedding = float32ToBase64(embedding)
			}
		}
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

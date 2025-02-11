package handler

import (
	"encoding/json"
	"net/http"

	"github.com/yammerjp/cachembed/internal/upstream"
)

// HandlerError はハンドラーのエラー情報を保持する構造体
type HandlerError struct {
	Status       int    // HTTPステータスコード
	Message      string // クライアントに返すエラーメッセージ
	ErrorType    string // エラータイプ（"invalid_request_error"など）
	InternalErr  error  // 内部エラー（ログ用）
	PromptTokens int    // 使用したトークン数（エラー時でも記録が必要な場合）
	TotalTokens  int    // 使用した合計トークン数
}

func (e *HandlerError) Error() string {
	if e.InternalErr != nil {
		return e.InternalErr.Error()
	}
	return e.Message
}

// NewHandlerError は新しいHandlerErrorを作成します（トークン情報なし）
func NewHandlerError(status int, message, errorType string, err error) *HandlerError {
	return NewHandlerErrorWithTokens(status, message, errorType, err, 0, 0)
}

// NewHandlerErrorWithTokens は新しいHandlerErrorを作成します（トークン情報あり）
func NewHandlerErrorWithTokens(status int, message, errorType string, err error, promptTokens, totalTokens int) *HandlerError {
	return &HandlerError{
		Status:       status,
		Message:      message,
		ErrorType:    errorType,
		InternalErr:  err,
		PromptTokens: promptTokens,
		TotalTokens:  totalTokens,
	}
}

// WriteResponse はエラーレスポンスを書き込みます
func (e *HandlerError) WriteResponse(w http.ResponseWriter) {
	writeError(w, e.Status, e.Message, e.ErrorType)
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

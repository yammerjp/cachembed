package handler

import (
	"encoding/json"
	"net/http"

	"github.com/yammerjp/cachembed/internal/upstream"
)

func writeError(w http.ResponseWriter, status int, message, errType string) {
	var resp upstream.ErrorResponse
	resp.Error.Message = message
	resp.Error.Type = errType
	resp.Error.Code = http.StatusText(status)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(resp)
}

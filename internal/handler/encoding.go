package handler

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"log/slog"
	"math"
)

func float32ToBase64(values []float32) string {
	buf := new(bytes.Buffer)
	for _, v := range values {
		bits := math.Float32bits(v)
		buf.Write([]byte{byte(bits), byte(bits >> 8), byte(bits >> 16), byte(bits >> 24)})
	}
	return base64.StdEncoding.EncodeToString(buf.Bytes())
}

func formatEmbedding(embedding []float32, format string) interface{} {
	if format == "base64" {
		return float32ToBase64(embedding)
	}
	return embedding
}

func convertToFloat32Slice(v interface{}) ([]float32, bool) {
	slog.Debug("converting embedding",
		"type", fmt.Sprintf("%T", v),
		"value", fmt.Sprintf("%v", v),
	)

	switch x := v.(type) {
	case []float32:
		slog.Debug("found float32 slice")
		return x, true
	case []float64:
		slog.Debug("found float64 slice")
		result := make([]float32, len(x))
		for i, val := range x {
			result[i] = float32(val)
		}
		return result, true
	case []interface{}:
		slog.Debug("found interface slice")
		result := make([]float32, len(x))
		for i, val := range x {
			switch v := val.(type) {
			case float64:
				result[i] = float32(v)
			case float32:
				result[i] = v
			default:
				slog.Debug("invalid type in interface slice",
					"index", i,
					"type", fmt.Sprintf("%T", val),
				)
				return nil, false
			}
		}
		return result, true
	default:
		slog.Debug("unknown type")
		return nil, false
	}
}

func base64ToFloat32Slice(b64 string) ([]float32, error) {
	data, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		return nil, fmt.Errorf("failed to decode base64: %w", err)
	}

	if len(data)%4 != 0 {
		return nil, fmt.Errorf("invalid data length: %d", len(data))
	}

	result := make([]float32, len(data)/4)
	for i := 0; i < len(data); i += 4 {
		bits := uint32(data[i]) |
			uint32(data[i+1])<<8 |
			uint32(data[i+2])<<16 |
			uint32(data[i+3])<<24
		result[i/4] = math.Float32frombits(bits)
	}
	return result, nil
}

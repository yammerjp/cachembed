package util

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"math"
)

func Float32ToBase64(values EmbeddedVectorFloat32) (EmbeddedVectorBase64, error) {
	buf := new(bytes.Buffer)
	for _, v := range values {
		bits := math.Float32bits(v)
		buf.Write([]byte{byte(bits), byte(bits >> 8), byte(bits >> 16), byte(bits >> 24)})
	}
	return EmbeddedVectorBase64(base64.StdEncoding.EncodeToString(buf.Bytes())), nil
}

func Base64ToFloat32Slice(b64 EmbeddedVectorBase64) (EmbeddedVectorFloat32, error) {
	data, err := base64.StdEncoding.DecodeString(string(b64))
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
	return EmbeddedVectorFloat32(result), nil
}

package upstream

import (
	"bytes"
	"encoding/base64"
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

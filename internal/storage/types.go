package storage

import (
	"github.com/yammerjp/cachembed/internal/types"
)

// Database はストレージのインターフェースです
// dimension が 0 の場合はデフォルト値として扱われます
type Database interface {
	StoreEmbedding(hash string, model string, dimension int, embeddingBase64 types.EmbeddedVectorBase64) error
	GetEmbedding(hash string, model string, dimension int) (types.EmbeddedVectorBase64, error)
}

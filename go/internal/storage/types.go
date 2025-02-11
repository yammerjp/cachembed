package storage

import "github.com/yammerjp/cachembed/internal/util"

// Database はストレージのインターフェースです
// dimension が 0 の場合はデフォルト値として扱われます
type Database interface {
	StoreEmbedding(hash string, model string, dimension int, embeddingBase64 util.EmbeddedVectorBase64) error
	GetEmbedding(hash string, model string, dimension int) (util.EmbeddedVectorBase64, error)
}

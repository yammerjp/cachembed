package storage

// Database はストレージのインターフェースです
type Database interface {
	StoreEmbedding(hash string, model string, embedding []float32) error
	GetEmbedding(hash string, model string) ([]float32, error)
}

package storage

// MockDB はテスト用のモックデータベース
type MockDB struct {
	embeddings map[string][]float32
}

func NewMockDB() *MockDB {
	return &MockDB{
		embeddings: make(map[string][]float32),
	}
}

func (db *MockDB) GetEmbedding(hash, model string) ([]float32, error) {
	key := hash + ":" + model
	if embedding, ok := db.embeddings[key]; ok {
		return embedding, nil
	}
	return nil, nil
}

func (db *MockDB) StoreEmbedding(hash, model string, embedding []float32) error {
	key := hash + ":" + model
	db.embeddings[key] = embedding
	return nil
}

// MockDB が Database インターフェースを実装していることを確認
var _ Database = (*MockDB)(nil)

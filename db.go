package main

import (
	"bytes"
	"database/sql"
	"encoding/binary"
	"fmt"
	"log/slog"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

const schema = `
CREATE TABLE IF NOT EXISTS embeddings (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    input_hash TEXT NOT NULL,
    model TEXT NOT NULL,
    embedding_data BLOB NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    last_accessed_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(input_hash, model)
);

CREATE INDEX IF NOT EXISTS idx_last_accessed ON embeddings(last_accessed_at);
CREATE INDEX IF NOT EXISTS idx_input_model ON embeddings(input_hash, model);
`

type DB struct {
	*sql.DB
}

// EmbeddingCache はキャッシュされた埋め込みデータを表します
type EmbeddingCache struct {
	EmbeddingData []float32
	CreatedAt     time.Time
	LastAccessed  time.Time
}

func NewDB(dsn string) (*DB, error) {
	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	// Enable WAL mode for better concurrency
	if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		return nil, fmt.Errorf("failed to enable WAL mode: %w", err)
	}

	// Create schema
	if _, err := db.Exec(schema); err != nil {
		return nil, fmt.Errorf("failed to create schema: %w", err)
	}

	slog.Info("database initialized", "dsn", dsn)
	return &DB{db}, nil
}

func (db *DB) Close() error {
	return db.DB.Close()
}

// GetEmbedding は指定されたinput_hashとmodelの埋め込みを取得します
func (db *DB) GetEmbedding(inputHash, model string) (*EmbeddingCache, error) {
	var cache EmbeddingCache
	var blobData []byte

	err := db.QueryRow(`
		UPDATE embeddings 
		SET last_accessed_at = CURRENT_TIMESTAMP 
		WHERE input_hash = ? AND model = ? 
		RETURNING embedding_data, created_at, last_accessed_at
	`, inputHash, model).Scan(&blobData, &cache.CreatedAt, &cache.LastAccessed)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get embedding: %w", err)
	}

	// BLOBからfloat32スライスに変換
	cache.EmbeddingData = make([]float32, len(blobData)/4)
	if err := binary.Read(bytes.NewReader(blobData), binary.LittleEndian, &cache.EmbeddingData); err != nil {
		return nil, fmt.Errorf("failed to decode embedding data: %w", err)
	}

	return &cache, nil
}

// StoreEmbedding は埋め込みデータをキャッシュに保存します
func (db *DB) StoreEmbedding(inputHash, model string, embedding []float32) error {
	// float32スライスをBLOBに変換
	buf := new(bytes.Buffer)
	if err := binary.Write(buf, binary.LittleEndian, embedding); err != nil {
		return fmt.Errorf("failed to encode embedding data: %w", err)
	}

	_, err := db.Exec(`
		INSERT INTO embeddings (input_hash, model, embedding_data) 
		VALUES (?, ?, ?)
		ON CONFLICT(input_hash, model) DO UPDATE 
		SET embedding_data = excluded.embedding_data,
		    last_accessed_at = CURRENT_TIMESTAMP
	`, inputHash, model, buf.Bytes())

	if err != nil {
		return fmt.Errorf("failed to store embedding: %w", err)
	}

	return nil
}

// DeleteOldEntries はLRUキャッシュのガベージコレクションを実行します
func (db *DB) DeleteOldEntries(limit int) error {
	_, err := db.Exec(`
		DELETE FROM embeddings 
		WHERE id IN (
			SELECT id FROM embeddings 
			ORDER BY last_accessed_at ASC 
			LIMIT ?
		)
	`, limit)

	if err != nil {
		return fmt.Errorf("failed to delete old entries: %w", err)
	}

	return nil
}

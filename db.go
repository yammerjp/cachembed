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

// runMigrations はデータベースのマイグレーションを実行します
func runMigrations(db *sql.DB) error {
	// スキーマを直接実行
	if _, err := db.Exec(schema); err != nil {
		return fmt.Errorf("failed to create schema: %w", err)
	}
	return nil
}

func NewDB(dsn string) (*DB, error) {
	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Enable WAL mode for better concurrency
	if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		return nil, fmt.Errorf("failed to enable WAL mode: %w", err)
	}

	// Run migrations
	if err := runMigrations(db); err != nil {
		return nil, fmt.Errorf("failed to run migrations: %w", err)
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

	tx, err := db.Begin()
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	err = tx.QueryRow(`
		SELECT embedding_data, created_at, last_accessed_at
		FROM embeddings 
		WHERE input_hash = ? AND model = ?
	`, inputHash, model).Scan(&blobData, &cache.CreatedAt, &cache.LastAccessed)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get embedding: %w", err)
	}

	// アクセス時刻を更新
	_, err = tx.Exec(`
		UPDATE embeddings
		SET last_accessed_at = CURRENT_TIMESTAMP
		WHERE input_hash = ? AND model = ?
	`, inputHash, model)
	if err != nil {
		return nil, fmt.Errorf("failed to update last_accessed_at: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
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

	// embeddingsテーブルに挿入または更新
	_, err := db.Exec(`
		INSERT INTO embeddings (input_hash, model, embedding_data, last_accessed_at) 
		VALUES (?, ?, ?, CURRENT_TIMESTAMP)
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
	result, err := db.Exec(`
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

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get affected rows: %w", err)
	}

	slog.Info("garbage collection completed",
		"deleted_entries", rowsAffected,
		"requested_limit", limit,
	)

	return nil
}

// DeleteEntriesBefore は指定された期間より前にアクセスされたエントリを削除します
func (db *DB) DeleteEntriesBefore(age time.Duration, limit int) error {
	query := `
		DELETE FROM embeddings
		WHERE last_accessed_at < datetime('now', ?)
	`
	args := []interface{}{fmt.Sprintf("-%d seconds", int(age.Seconds()))}

	if limit > 0 {
		query = `
			DELETE FROM embeddings
			WHERE id IN (
				SELECT id FROM embeddings
				WHERE last_accessed_at < datetime('now', ?)
				ORDER BY last_accessed_at ASC
				LIMIT ?
			)
		`
		args = append(args, limit)
	}

	result, err := db.Exec(query, args...)
	if err != nil {
		return fmt.Errorf("failed to delete old entries: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get affected rows: %w", err)
	}

	slog.Info("garbage collection completed",
		"deleted_entries", rowsAffected,
		"age", age,
		"limit", limit,
	)

	return nil
}

// ヘルパー関数
func interfaceSlice(slice []int64) []interface{} {
	interfaces := make([]interface{}, len(slice))
	for i, v := range slice {
		interfaces[i] = v
	}
	return interfaces
}

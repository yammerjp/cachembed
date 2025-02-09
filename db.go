package main

import (
	"bytes"
	"context"
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

// Sleeper はスリープ機能を抽象化するインターフェース
type Sleeper interface {
	Sleep(d time.Duration)
}

// RealSleeper は実際のtime.Sleepを使用する実装
type RealSleeper struct{}

func (s RealSleeper) Sleep(d time.Duration) {
	time.Sleep(d)
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

// DB構造体にSleeperを追加
type DB struct {
	*sql.DB
	sleeper Sleeper
}

// NewDB関数を修正
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
	return &DB{DB: db, sleeper: RealSleeper{}}, nil
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

// DeleteEntriesBefore は指定された期間より前にアクセスされたエントリを削除します
// startID から endID までの範囲で指定された期間より前のエントリを削除します
func (db *DB) DeleteEntriesBefore(age time.Duration, startID, endID int64) error {
	const batchSize = 1000
	var totalDeleted int64
	currentID := startID

	ageSeconds := fmt.Sprintf("-%d seconds", int(age.Seconds()))

	for currentID < endID {
		// バッチの範囲を決定
		batchEndID := currentID + batchSize
		if batchEndID > endID {
			batchEndID = endID
		}

		// 指定範囲のレコードを削除
		query := `
			DELETE FROM embeddings
			WHERE id >= ? AND id < ?
			AND last_accessed_at < datetime('now', ?)
		`
		result, err := db.Exec(query, currentID, batchEndID, ageSeconds)
		if err != nil {
			return fmt.Errorf("failed to delete batch: %w", err)
		}

		rowsAffected, err := result.RowsAffected()
		if err != nil {
			return fmt.Errorf("failed to get affected rows: %w", err)
		}

		totalDeleted += rowsAffected

		// 進捗をログに出力
		slog.Info("batch deletion progress",
			"current_id", currentID,
			"batch_end_id", batchEndID,
			"batch_deleted", rowsAffected,
			"total_deleted", totalDeleted,
		)

		currentID = batchEndID
	}

	slog.Info("garbage collection completed",
		"deleted_entries", totalDeleted,
		"age", age,
		"start_id", startID,
		"end_id", endID,
	)

	return nil
}

// DeleteEntriesBeforeWithSleep を修正
func (db *DB) DeleteEntriesBeforeWithSleep(age time.Duration, startID, endID int64, sleep time.Duration) error {
	const batchSize = 1000
	var totalDeleted int64
	currentID := startID

	ageSeconds := fmt.Sprintf("-%d seconds", int(age.Seconds()))

	for currentID < endID {
		// バッチの範囲を決定
		batchEndID := currentID + batchSize
		if batchEndID > endID {
			batchEndID = endID
		}

		// 指定範囲のレコードを削除
		query := `
			DELETE FROM embeddings
			WHERE id >= ? AND id < ?
			AND last_accessed_at < datetime('now', ?)
		`
		result, err := db.Exec(query, currentID, batchEndID, ageSeconds)
		if err != nil {
			return fmt.Errorf("failed to delete batch: %w", err)
		}

		rowsAffected, err := result.RowsAffected()
		if err != nil {
			return fmt.Errorf("failed to get affected rows: %w", err)
		}

		totalDeleted += rowsAffected

		// 進捗をログに出力
		slog.Info("batch deletion progress",
			"current_id", currentID,
			"batch_end_id", batchEndID,
			"batch_deleted", rowsAffected,
			"total_deleted", totalDeleted,
		)

		// スリープ処理を修正
		if sleep > 0 {
			db.sleeper.Sleep(sleep)
		}

		currentID = batchEndID
	}

	slog.Info("garbage collection completed",
		"deleted_entries", totalDeleted,
		"age", age,
		"start_id", startID,
		"end_id", endID,
	)

	return nil
}

// GetMaxID は現在のデータベース内の最大IDを返します
func (d *DB) GetMaxID(ctx context.Context) (int64, error) {
	var maxID int64
	query := "SELECT MAX(id) FROM items"
	err := d.DB.QueryRowContext(ctx, query).Scan(&maxID)
	if err != nil {
		return 0, fmt.Errorf("failed to get max id: %w", err)
	}
	return maxID, nil
}

// DeleteByID は指定されたIDのエントリを削除します
func (db *DB) DeleteByID(ctx context.Context, id int64) error {
	query := `DELETE FROM embeddings WHERE id = ?`
	result, err := db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete entry with id %d: %w", id, err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get affected rows: %w", err)
	}

	if rowsAffected > 0 {
		slog.Debug("deleted entry", "id", id)
	}

	return nil
}

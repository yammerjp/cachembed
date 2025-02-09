package main

import (
	"bytes"
	"database/sql"
	"encoding/binary"
	"fmt"
	"log/slog"
	"net/url"
	"strings"
	"time"

	_ "github.com/lib/pq"           // PostgreSQLドライバー
	_ "github.com/mattn/go-sqlite3" // SQLite3ドライバー
)

// Dialect はデータベース固有のSQL文を提供するインターフェース
type Dialect interface {
	// GetPrimaryKeyType はデータベース固有のプライマリキー型を返します
	GetPrimaryKeyType() string
	// GetBlobType はデータベース固有のBLOB型を返します
	GetBlobType() string
}

// SQLiteDialect はSQLite用の実装
type SQLiteDialect struct{}

func (d SQLiteDialect) GetPrimaryKeyType() string {
	return "INTEGER PRIMARY KEY AUTOINCREMENT"
}

func (d SQLiteDialect) GetBlobType() string {
	return "BLOB"
}

// PostgreSQLDialect はPostgreSQL用の実装
type PostgreSQLDialect struct{}

func (d PostgreSQLDialect) GetPrimaryKeyType() string {
	return "BIGSERIAL PRIMARY KEY"
}

func (d PostgreSQLDialect) GetBlobType() string {
	return "BYTEA"
}

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
func runMigrations(db *sql.DB, dialect Dialect) error {
	// テーブルの作成
	createTableSQL := fmt.Sprintf(sqlCreateTable,
		dialect.GetPrimaryKeyType(),
		dialect.GetBlobType())

	if _, err := db.Exec(createTableSQL); err != nil {
		return fmt.Errorf("failed to create schema: %w", err)
	}

	// インデックスの作成
	createIndexSQL := `
	CREATE INDEX IF NOT EXISTS idx_input_model 
	ON embeddings(input_hash, model)
	`
	if _, err := db.Exec(createIndexSQL); err != nil {
		return fmt.Errorf("failed to create index: %w", err)
	}

	return nil
}

// DBConfig はデータベース設定を保持します
type DBConfig struct {
	Driver  string
	DSN     string
	Dialect Dialect
}

// ParseDSN はDSN文字列からデータベース設定を解析します
func ParseDSN(dsn string) (*DBConfig, error) {
	// SQLiteの場合
	if strings.HasSuffix(dsn, ".db") || strings.HasPrefix(dsn, "file:") || strings.HasPrefix(dsn, ":memory:") {
		return &DBConfig{
			Driver:  "sqlite3",
			DSN:     dsn,
			Dialect: SQLiteDialect{},
		}, nil
	}

	// URLベースのDSNをパース
	u, err := url.Parse(dsn)
	if err != nil {
		return nil, fmt.Errorf("invalid DSN format: %w", err)
	}

	switch u.Scheme {
	case "postgres", "postgresql":
		return &DBConfig{
			Driver:  "postgres",
			DSN:     dsn,
			Dialect: PostgreSQLDialect{},
		}, nil
	default:
		return nil, fmt.Errorf("unsupported database type: %s (only sqlite3 and postgres are supported)", u.Scheme)
	}
}

// DB構造体を修正
type DB struct {
	*sql.DB
	sleeper Sleeper
	dialect Dialect
	driver  string // プレースホルダー変換用
}

// NewDB関数を修正
func NewDB(dsn string) (*DB, error) {
	config, err := ParseDSN(dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to parse DSN: %w", err)
	}

	db, err := sql.Open(config.Driver, config.DSN)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// 接続テスト
	if err := db.Ping(); err != nil {
		db.Close() // エラー時にはDBをクローズ
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	// SQLite固有の設定
	if config.Driver == "sqlite3" {
		if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
			db.Close()
			return nil, fmt.Errorf("failed to enable WAL mode: %w", err)
		}
	}

	// マイグレーションを実行
	if err := runMigrations(db, config.Dialect); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	return &DB{
		DB:      db,
		sleeper: RealSleeper{},
		dialect: config.Dialect,
		driver:  config.Driver,
	}, nil
}

func (db *DB) Close() error {
	return db.DB.Close()
}

// プレースホルダー変換関数を修正
func (db *DB) convertPlaceholders(query string) string {
	if db.driver == "postgres" {
		// PostgreSQLでは$1, $2, ...の形式を使用
		parts := strings.Split(query, "?")
		if len(parts) == 1 {
			return query
		}
		var sb strings.Builder
		for i := 0; i < len(parts)-1; i++ {
			sb.WriteString(parts[i])
			sb.WriteString(fmt.Sprintf("$%d", i+1))
		}
		sb.WriteString(parts[len(parts)-1])
		return sb.String()
	}
	// SQLiteでは?を使用
	return query
}

// batchSizeを定数として定義
const (
	batchSize = 1000 // 一度に削除する最大レコード数

	// テーブル作成用のSQL
	sqlCreateTable = `
	CREATE TABLE IF NOT EXISTS embeddings (
		id %s,
		input_hash TEXT NOT NULL,
		model TEXT NOT NULL,
		embedding_data %s NOT NULL,
		created_at TIMESTAMP NOT NULL,
		last_accessed_at TIMESTAMP NOT NULL,
		UNIQUE(input_hash, model)
	)`

	// 共通のクエリ
	sqlGetEmbedding = `
	SELECT embedding_data, created_at, last_accessed_at
	FROM embeddings 
	WHERE input_hash = ? AND model = ?`

	sqlUpdateLastAccessed = `
	UPDATE embeddings
	SET last_accessed_at = ?
	WHERE input_hash = ? AND model = ?`

	sqlStoreEmbedding = `
	INSERT INTO embeddings (input_hash, model, embedding_data, created_at, last_accessed_at) 
	VALUES (?, ?, ?, ?, ?)
	ON CONFLICT(input_hash, model) DO UPDATE 
	SET embedding_data = excluded.embedding_data,
		last_accessed_at = excluded.last_accessed_at`
)

// GetEmbedding を修正
func (db *DB) GetEmbedding(inputHash, model string) (*EmbeddingCache, error) {
	var cache EmbeddingCache
	var blobData []byte

	tx, err := db.Begin()
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	query := db.convertPlaceholders(sqlGetEmbedding)
	err = tx.QueryRow(query, inputHash, model).Scan(&blobData, &cache.CreatedAt, &cache.LastAccessed)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get embedding: %w", err)
	}

	// アクセス時刻を更新
	now := time.Now().UTC()
	updateQuery := db.convertPlaceholders(`
		UPDATE embeddings
		SET last_accessed_at = ?
		WHERE input_hash = ? AND model = ?
	`)
	_, err = tx.Exec(updateQuery, now, inputHash, model)
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

// StoreEmbedding を修正
func (db *DB) StoreEmbedding(inputHash, model string, embedding []float32) error {
	// float32スライスをBLOBに変換
	buf := new(bytes.Buffer)
	if err := binary.Write(buf, binary.LittleEndian, embedding); err != nil {
		return fmt.Errorf("failed to encode embedding data: %w", err)
	}

	// 現在時刻をUTCで取得
	now := time.Now().UTC()

	// embeddingsテーブルに挿入または更新
	query := db.convertPlaceholders(`
		INSERT INTO embeddings (input_hash, model, embedding_data, created_at, last_accessed_at) 
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(input_hash, model) DO UPDATE 
		SET embedding_data = excluded.embedding_data,
			last_accessed_at = excluded.last_accessed_at
	`)
	// バイナリデータはそのまま[]byteとして渡す
	_, err := db.Exec(query, inputHash, model, buf.Bytes(), now, now)
	if err != nil {
		return fmt.Errorf("failed to store embedding: %w", err)
	}

	return nil
}

func (db *DB) DeleteEntriesBeforeWithSleep(threshold time.Duration, startID, endID int64, sleep time.Duration) error {
	// 現在時刻から threshold を引いた時刻を計算（UTCで）
	thresholdTime := time.Now().UTC().Add(-threshold)

	// SQLを統一
	query := db.convertPlaceholders(`
		DELETE FROM embeddings
		WHERE id >= ? AND id < ?
		AND last_accessed_at < ?
	`)

	var totalDeleted int64
	currentID := startID

	for currentID < endID {
		batchEndID := currentID + batchSize - 1
		if batchEndID >= endID {
			batchEndID = endID - 1
		}

		result, err := db.Exec(query, currentID, batchEndID+1, thresholdTime)
		if err != nil {
			return fmt.Errorf("failed to delete batch: %w", err)
		}

		rowsAffected, err := result.RowsAffected()
		if err != nil {
			return fmt.Errorf("failed to get affected rows: %w", err)
		}

		totalDeleted += rowsAffected

		slog.Info("batch deletion progress",
			"current_id", currentID,
			"batch_end_id", batchEndID,
			"batch_deleted", rowsAffected,
			"total_deleted", totalDeleted,
			"threshold_time", thresholdTime)

		if sleep > 0 {
			db.sleeper.Sleep(sleep)
		}

		currentID = batchEndID + 1
	}

	return nil
}

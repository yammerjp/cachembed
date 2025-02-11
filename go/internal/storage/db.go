package storage

import (
	"database/sql"
	"fmt"
	"log/slog"
	"time"

	"github.com/yammerjp/cachembed/internal/util"
)

const (
	sqlCreateTable = `
	CREATE TABLE IF NOT EXISTS embeddings (
		id %s,
		input_hash TEXT NOT NULL,
		model TEXT NOT NULL,
		dimension INTEGER DEFAULT 0 NOT NULL, -- dimension of 0 indicates the API's default dimension size
		embedding_data TEXT NOT NULL, -- base64 encoded float array
		created_at TIMESTAMP NOT NULL,
		last_accessed_at TIMESTAMP NOT NULL,
		UNIQUE(input_hash, model, dimension)
	)`

	createIndexSQL = `
	CREATE INDEX IF NOT EXISTS idx_input_model_dim 
	ON embeddings(input_hash, model, dimension)
	`

	sqlGetEmbedding = `
	SELECT embedding_data
	FROM embeddings 
	WHERE input_hash = $1 AND model = $2 AND dimension = $3`

	sqlUpdateLastAccessed = `
	UPDATE embeddings
	SET last_accessed_at = $1
	WHERE input_hash = $2 AND model = $3 AND dimension = $4`

	sqlStoreEmbedding = `
	INSERT INTO embeddings (input_hash, model, dimension, embedding_data, created_at, last_accessed_at) 
	VALUES ($1, $2, $3, $4, $5, $6)
	ON CONFLICT(input_hash, model, dimension) DO UPDATE 
	SET embedding_data = EXCLUDED.embedding_data,
		last_accessed_at = EXCLUDED.last_accessed_at`

	sqlDeleteEntriesBefore = `
		DELETE FROM embeddings
		WHERE id >= $1 AND id < $2
		AND last_accessed_at < $3
	`

	sqlGetMaxID = `
		SELECT COALESCE(MAX(id), 0) FROM embeddings
	`
)

type EmbeddingCache struct {
	EmbeddingData string
	CreatedAt     time.Time
	LastAccessed  time.Time
}

type DB struct {
	*sql.DB
	sleeper Sleeper
	dialect Dialect
}

func (db *DB) Exec(query string, args ...interface{}) (sql.Result, error) {
	query = db.dialect.ConvertPlaceholders(query)
	return db.DB.Exec(query, args...)
}

func NewDB(dsn string) (*DB, error) {
	config, err := parseDSN(dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to parse DSN: %w", err)
	}

	db, err := sql.Open(config.Driver, config.DSN)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	if err := config.Dialect.Initialize(db); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to initialize database: %w", err)
	}

	ret := &DB{
		DB:      db,
		sleeper: RealSleeper{},
		dialect: config.Dialect,
	}

	if err := ret.RunMigrations(); err != nil {
		ret.Close()
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	return ret, nil
}

func (db *DB) Close() error {
	return db.DB.Close()
}

func (db *DB) RunMigrations() error {
	createTableSQL := fmt.Sprintf(sqlCreateTable,
		db.dialect.GetPrimaryKeyType())

	if _, err := db.Exec(createTableSQL); err != nil {
		return fmt.Errorf("failed to create schema: %w", err)
	}

	if _, err := db.Exec(createIndexSQL); err != nil {
		return fmt.Errorf("failed to create index: %w", err)
	}
	return nil
}

func (db *DB) GetEmbedding(hash string, model string, dimension int) (util.EmbeddedVectorBase64, error) {
	var embeddingBase64 util.EmbeddedVectorBase64
	err := db.QueryRow(
		sqlGetEmbedding,
		hash,
		model,
		dimension,
	).Scan(&embeddingBase64)

	if err == sql.ErrNoRows {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("failed to query embedding: %w", err)
	}

	// 最終アクセス時刻を更新
	_, err = db.Exec(
		sqlUpdateLastAccessed,
		time.Now().UTC(),
		hash,
		model,
		dimension,
	)
	if err != nil {
		slog.Error("failed to update last accessed time", "error", err)
	}

	return embeddingBase64, nil
}

func (db *DB) StoreEmbedding(inputHash, model string, dimension int, embeddingBase64 util.EmbeddedVectorBase64) error {
	now := time.Now().UTC()

	query := db.dialect.ConvertPlaceholders(sqlStoreEmbedding)
	_, err := db.Exec(query, inputHash, model, dimension, embeddingBase64, now, now)
	if err != nil {
		return fmt.Errorf("failed to store embedding: %w", err)
	}

	return nil
}

func (db *DB) DeleteEntriesBeforeWithSleep(threshold time.Duration, startID, endID int64, batchSize int64, sleep time.Duration) error {
	thresholdTime := time.Now().UTC().Add(-threshold)

	query := db.dialect.ConvertPlaceholders(sqlDeleteEntriesBefore)

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

func (db *DB) GetMaxID() (int64, error) {
	var maxID int64
	err := db.QueryRow(sqlGetMaxID).Scan(&maxID)
	if err != nil {
		return 0, fmt.Errorf("failed to get max ID: %w", err)
	}
	return maxID, nil
}

package storage

import (
	"bytes"
	"database/sql"
	"encoding/binary"
	"fmt"
	"log/slog"
	"time"
)

const (
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

	createIndexSQL = `
	CREATE INDEX IF NOT EXISTS idx_input_model 
	ON embeddings(input_hash, model)
	`

	sqlGetEmbedding = `
	SELECT embedding_data, created_at, last_accessed_at
	FROM embeddings 
	WHERE input_hash = $1 AND model = $2`

	sqlUpdateLastAccessed = `
	UPDATE embeddings
	SET last_accessed_at = $1
	WHERE input_hash = $2 AND model = $3`

	sqlStoreEmbedding = `
	INSERT INTO embeddings (input_hash, model, embedding_data, created_at, last_accessed_at) 
	VALUES ($1, $2, $3, $4, $5)
	ON CONFLICT(input_hash, model) DO UPDATE 
	SET embedding_data = excluded.embedding_data,
		last_accessed_at = excluded.last_accessed_at`

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
	EmbeddingData []float32
	CreatedAt     time.Time
	LastAccessed  time.Time
}

type DB struct {
	*sql.DB
	sleeper Sleeper
	dialect Dialect
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
		db.dialect.GetPrimaryKeyType(),
		db.dialect.GetBlobType())

	if _, err := db.Exec(createTableSQL); err != nil {
		return fmt.Errorf("failed to create schema: %w", err)
	}

	if _, err := db.Exec(createIndexSQL); err != nil {
		return fmt.Errorf("failed to create index: %w", err)
	}
	return nil
}

func (db *DB) GetEmbedding(inputHash, model string) (*EmbeddingCache, error) {
	var cache EmbeddingCache
	var blobData []byte

	tx, err := db.Begin()
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	query := db.dialect.ConvertPlaceholders(sqlGetEmbedding)
	err = tx.QueryRow(query, inputHash, model).Scan(&blobData, &cache.CreatedAt, &cache.LastAccessed)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get embedding: %w", err)
	}

	now := time.Now().UTC()
	updateQuery := db.dialect.ConvertPlaceholders(sqlUpdateLastAccessed)
	_, err = tx.Exec(updateQuery, now, inputHash, model)
	if err != nil {
		return nil, fmt.Errorf("failed to update last_accessed_at: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	cache.EmbeddingData = make([]float32, len(blobData)/4)
	if err := binary.Read(bytes.NewReader(blobData), binary.LittleEndian, &cache.EmbeddingData); err != nil {
		return nil, fmt.Errorf("failed to decode embedding data: %w", err)
	}

	return &cache, nil
}

func (db *DB) StoreEmbedding(inputHash, model string, embedding []float32) error {
	buf := new(bytes.Buffer)
	if err := binary.Write(buf, binary.LittleEndian, embedding); err != nil {
		return fmt.Errorf("failed to encode embedding data: %w", err)
	}

	now := time.Now().UTC()

	query := db.dialect.ConvertPlaceholders(sqlStoreEmbedding)
	_, err := db.Exec(query, inputHash, model, buf.Bytes(), now, now)
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

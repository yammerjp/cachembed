package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"os"
	"testing"
	"time"
)

func TestNewDB(t *testing.T) {
	// Create temporary database file
	tmpFile, err := os.CreateTemp("", "cachembed-test-*.db")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	tests := []struct {
		name    string
		dsn     string
		wantErr bool
	}{
		{
			name:    "valid DSN creates database",
			dsn:     tmpFile.Name(),
			wantErr: false,
		},
		{
			name:    "invalid DSN returns error",
			dsn:     "/nonexistent/path/db.sqlite",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, err := NewDB(tt.dsn)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewDB() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil {
				return
			}
			defer db.Close()

			// Verify schema
			var tables []string
			rows, err := db.Query(`
				SELECT name FROM sqlite_master 
				WHERE type='table' AND name='embeddings'
			`)
			if err != nil {
				t.Fatalf("Failed to query tables: %v", err)
			}
			defer rows.Close()

			for rows.Next() {
				var name string
				if err := rows.Scan(&name); err != nil {
					t.Fatalf("Failed to scan row: %v", err)
				}
				tables = append(tables, name)
			}

			if len(tables) != 1 {
				t.Error("Expected table embeddings to exist")
			}

			// Verify indices
			var indices []string
			rows, err = db.Query(`
				SELECT name FROM sqlite_master 
				WHERE type='index' AND tbl_name='embeddings'
			`)
			if err != nil {
				t.Fatalf("Failed to query indices: %v", err)
			}
			defer rows.Close()

			for rows.Next() {
				var name string
				if err := rows.Scan(&name); err != nil {
					t.Fatalf("Failed to scan row: %v", err)
				}
				indices = append(indices, name)
			}

			expectedIndices := []string{"idx_input_model"}
			for _, idx := range expectedIndices {
				found := false
				for _, actual := range indices {
					if actual == idx {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected index %s to exist", idx)
				}
			}
		})
	}
}

func TestEmbeddingCacheOperations(t *testing.T) {
	// テスト用の一時データベースを作成
	tmpFile, err := os.CreateTemp("", "cachembed-test-*.db")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	db, err := NewDB(tmpFile.Name())
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	// マイグレーションを実行
	if err := runMigrations(db.DB); err != nil {
		t.Fatalf("Failed to run migrations: %v", err)
	}

	// テストデータ
	inputHash := "testhash123"
	model := "test-model"
	embedding := []float32{0.1, 0.2, 0.3, 0.4, 0.5}

	// Store操作のテスト
	t.Run("store embedding", func(t *testing.T) {
		err := db.StoreEmbedding(inputHash, model, embedding)
		if err != nil {
			t.Fatalf("Failed to store embedding: %v", err)
		}
	})

	// Get操作のテスト
	t.Run("get embedding", func(t *testing.T) {
		cache, err := db.GetEmbedding(inputHash, model)
		if err != nil {
			t.Fatalf("Failed to get embedding: %v", err)
		}
		if cache == nil {
			t.Fatal("Expected cache hit, got cache miss")
		}

		// データの検証
		if len(cache.EmbeddingData) != len(embedding) {
			t.Errorf("Expected embedding length %d, got %d", len(embedding), len(cache.EmbeddingData))
		}
		for i, v := range embedding {
			if cache.EmbeddingData[i] != v {
				t.Errorf("Embedding mismatch at index %d: expected %f, got %f", i, v, cache.EmbeddingData[i])
			}
		}
	})

	// キャッシュミスのテスト
	t.Run("cache miss", func(t *testing.T) {
		cache, err := db.GetEmbedding("nonexistent", model)
		if err != nil {
			t.Fatalf("Failed to query nonexistent embedding: %v", err)
		}
		if cache != nil {
			t.Error("Expected cache miss, got cache hit")
		}
	})

	// GC操作のテスト
	t.Run("garbage collection", func(t *testing.T) {
		// 既存のデータをクリア
		_, err := db.Exec("DELETE FROM embeddings")
		if err != nil {
			t.Fatalf("Failed to clear embeddings: %v", err)
		}

		// 古いエントリを作成
		for i := 0; i < 5; i++ {
			hash := fmt.Sprintf("old_hash%d", i)
			buf := new(bytes.Buffer)
			if err := binary.Write(buf, binary.LittleEndian, embedding); err != nil {
				t.Fatalf("Failed to encode embedding data: %v", err)
			}

			_, err := db.Exec(`
				INSERT INTO embeddings (input_hash, model, embedding_data, last_accessed_at)
				VALUES (?, ?, ?, datetime('now', '-1 hour'))
			`, hash, model, buf.Bytes())
			if err != nil {
				t.Fatalf("Failed to create old entry: %v", err)
			}
		}

		// 新しいエントリを作成
		for i := 0; i < 5; i++ {
			hash := fmt.Sprintf("new_hash%d", i)
			if err := db.StoreEmbedding(hash, model, embedding); err != nil {
				t.Fatalf("Failed to store embedding: %v", err)
			}
		}

		// 30分以上前のエントリを削除
		duration := 30 * time.Minute
		var maxID int64
		err = db.QueryRow("SELECT COALESCE(MAX(id), 0) FROM embeddings").Scan(&maxID)
		if err != nil {
			t.Fatalf("Failed to get max ID: %v", err)
		}
		if err := db.DeleteEntriesBefore(duration, 0, maxID); err != nil {
			t.Fatalf("Failed to delete old entries: %v", err)
		}

		// 残りのエントリ数を確認
		var count int
		err = db.QueryRow("SELECT COUNT(*) FROM embeddings").Scan(&count)
		if err != nil {
			t.Fatalf("Failed to count remaining entries: %v", err)
		}

		if count != 5 {
			t.Errorf("Expected 5 entries after GC, got %d", count)
		}

		// 古いエントリが削除されたことを確認
		var oldCount int
		err = db.QueryRow(`
			SELECT COUNT(*) FROM embeddings
			WHERE last_accessed_at < datetime('now', '-30 minutes')
		`).Scan(&oldCount)
		if err != nil {
			t.Fatalf("Failed to count old entries: %v", err)
		}

		if oldCount != 0 {
			t.Errorf("Expected no old entries after GC, got %d", oldCount)
		}
	})
}

func TestDeleteOldEntries(t *testing.T) {
	// テスト用の一時データベースを作成
	tmpFile, err := os.CreateTemp("", "cachembed-test-*.db")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	db, err := NewDB(tmpFile.Name())
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	// テストデータ
	model := "test-model"
	embedding := []float32{0.1, 0.2, 0.3}

	// 複数のエントリを作成
	for i := 0; i < 10; i++ {
		hash := fmt.Sprintf("hash%d", i)
		if err := db.StoreEmbedding(hash, model, embedding); err != nil {
			t.Fatalf("Failed to store embedding: %v", err)
		}
	}

	// 最初の5つのエントリのアクセス時刻を1時間前に設定
	for i := 0; i < 5; i++ {
		hash := fmt.Sprintf("hash%d", i)
		_, err := db.Exec(`
			UPDATE embeddings 
			SET last_accessed_at = datetime('now', '-1 hour')
			WHERE input_hash = ?
		`, hash)
		if err != nil {
			t.Fatalf("Failed to update access time: %v", err)
		}
	}

	// GCを実行（古い5つのエントリを削除）
	duration := 30 * time.Minute
	var maxID int64
	err = db.QueryRow("SELECT COALESCE(MAX(id), 0) FROM embeddings").Scan(&maxID)
	if err != nil {
		t.Fatalf("Failed to get max ID: %v", err)
	}
	if err := db.DeleteEntriesBefore(duration, 0, maxID); err != nil {
		t.Fatalf("Failed to run garbage collection: %v", err)
	}

	// 残りのエントリ数を確認
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM embeddings").Scan(&count)
	if err != nil {
		t.Fatalf("Failed to count remaining entries: %v", err)
	}

	if count != 5 {
		t.Errorf("Expected 5 entries after GC, got %d", count)
	}

	// 古いエントリが削除されたことを確認
	var oldCount int
	err = db.QueryRow(`
		SELECT COUNT(*) FROM embeddings
		WHERE last_accessed_at < datetime('now', '-30 minutes')
	`).Scan(&oldCount)
	if err != nil {
		t.Fatalf("Failed to count old entries: %v", err)
	}

	if oldCount != 0 {
		t.Errorf("Expected no old entries after GC, got %d", oldCount)
	}
}

func TestDeleteEntriesBeforeWithIDRange(t *testing.T) {
	// テスト用の一時データベースを作成
	tmpFile, err := os.CreateTemp("", "cachembed-test-*.db")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	db, err := NewDB(tmpFile.Name())
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	// テストデータ
	model := "test-model"
	embedding := []float32{0.1, 0.2, 0.3}

	// 10個のエントリを作成
	for i := 0; i < 10; i++ {
		hash := fmt.Sprintf("hash%d", i)
		if err := db.StoreEmbedding(hash, model, embedding); err != nil {
			t.Fatalf("Failed to store embedding: %v", err)
		}
	}

	// 最初の5つのエントリのアクセス時刻を1時間前に設定
	for i := 0; i < 5; i++ {
		hash := fmt.Sprintf("hash%d", i)
		_, err := db.Exec(`
			UPDATE embeddings 
			SET last_accessed_at = datetime('now', '-1 hour')
			WHERE input_hash = ?
		`, hash)
		if err != nil {
			t.Fatalf("Failed to update access time: %v", err)
		}
	}

	// ID 1-3の範囲で古いエントリを削除
	duration := 30 * time.Minute
	if err := db.DeleteEntriesBefore(duration, 1, 4); err != nil {
		t.Fatalf("Failed to run garbage collection: %v", err)
	}

	// 削除されたエントリを確認
	var count int
	err = db.QueryRow(`
		SELECT COUNT(*) FROM embeddings
		WHERE id BETWEEN 1 AND 3
		AND last_accessed_at < datetime('now', '-30 minutes')
	`).Scan(&count)
	if err != nil {
		t.Fatalf("Failed to count deleted entries: %v", err)
	}

	if count != 0 {
		t.Errorf("Expected no old entries in range 1-3, got %d", count)
	}

	// 範囲外のエントリが残っていることを確認
	var totalCount int
	err = db.QueryRow("SELECT COUNT(*) FROM embeddings").Scan(&totalCount)
	if err != nil {
		t.Fatalf("Failed to count remaining entries: %v", err)
	}

	expectedCount := 7 // 10 - 3 (deleted in range 1-3)
	if totalCount != expectedCount {
		t.Errorf("Expected %d total entries, got %d", expectedCount, totalCount)
	}
}

// MockSleeper はテスト用のスリープモック
type MockSleeper struct {
	sleepCalls chan time.Duration
}

func NewMockSleeper() *MockSleeper {
	return &MockSleeper{
		sleepCalls: make(chan time.Duration, 10),
	}
}

func (s *MockSleeper) Sleep(d time.Duration) {
	s.sleepCalls <- d
}

func TestDeleteEntriesBeforeWithSleep(t *testing.T) {
	// テスト用の一時データベースを作成
	tmpFile, err := os.CreateTemp("", "cachembed-test-*.db")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	db, err := NewDB(tmpFile.Name())
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	// テストデータ
	model := "test-model"
	embedding := []float32{0.1, 0.2, 0.3}

	// 10個のエントリを作成
	for i := 0; i < 10; i++ {
		hash := fmt.Sprintf("hash%d", i)
		if err := db.StoreEmbedding(hash, model, embedding); err != nil {
			t.Fatalf("Failed to store embedding: %v", err)
		}
	}

	// 最初の5つのエントリのアクセス時刻を1時間前に設定
	for i := 0; i < 5; i++ {
		hash := fmt.Sprintf("hash%d", i)
		_, err := db.Exec(`
			UPDATE embeddings 
			SET last_accessed_at = datetime('now', '-1 hour')
			WHERE input_hash = ?
		`, hash)
		if err != nil {
			t.Fatalf("Failed to update access time: %v", err)
		}
	}

	// MockSleeperを作成
	mockSleeper := NewMockSleeper()
	db.sleeper = mockSleeper

	// GCを実行
	duration := 30 * time.Minute
	sleep := 1 * time.Second
	if err := db.DeleteEntriesBeforeWithSleep(duration, 1, 4, sleep); err != nil {
		t.Fatalf("Failed to run garbage collection: %v", err)
	}

	// スリープが呼ばれたことを確認
	select {
	case d := <-mockSleeper.sleepCalls:
		if d != sleep {
			t.Errorf("Expected sleep duration %v, got %v", sleep, d)
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("Expected sleep to be called")
	}

	// 削除されたエントリを確認
	var count int
	err = db.QueryRow(`
		SELECT COUNT(*) FROM embeddings
		WHERE id BETWEEN 1 AND 3
		AND last_accessed_at < datetime('now', '-30 minutes')
	`).Scan(&count)
	if err != nil {
		t.Fatalf("Failed to count deleted entries: %v", err)
	}

	if count != 0 {
		t.Errorf("Expected no old entries in range 1-3, got %d", count)
	}

	// 範囲外のエントリが残っていることを確認
	var totalCount int
	err = db.QueryRow("SELECT COUNT(*) FROM embeddings").Scan(&totalCount)
	if err != nil {
		t.Fatalf("Failed to count remaining entries: %v", err)
	}

	expectedCount := 7 // 10 - 3 (deleted in range 1-3)
	if totalCount != expectedCount {
		t.Errorf("Expected %d total entries, got %d", expectedCount, totalCount)
	}
}

package main

import (
	"fmt"
	"os"
	"testing"
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

			if len(tables) != 1 || tables[0] != "embeddings" {
				t.Error("Expected embeddings table to exist")
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

			expectedIndices := []string{"idx_last_accessed", "idx_input_model"}
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
		// 追加のエントリを作成
		for i := 0; i < 5; i++ {
			hash := fmt.Sprintf("hash%d", i)
			err := db.StoreEmbedding(hash, model, embedding)
			if err != nil {
				t.Fatalf("Failed to store embedding: %v", err)
			}
		}

		// 古いエントリを削除
		err := db.DeleteOldEntries(3)
		if err != nil {
			t.Fatalf("Failed to delete old entries: %v", err)
		}

		// 残りのエントリ数を確認
		var count int
		err = db.QueryRow("SELECT COUNT(*) FROM embeddings").Scan(&count)
		if err != nil {
			t.Fatalf("Failed to count remaining entries: %v", err)
		}

		if count != 3 {
			t.Errorf("Expected 3 entries after GC, got %d", count)
		}
	})
}

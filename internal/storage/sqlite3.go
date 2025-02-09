package storage

import (
	"database/sql"
	"fmt"
	"regexp"

	_ "github.com/mattn/go-sqlite3"
)

// SQLiteDialect はSQLite用の実装
type SQLiteDialect struct{}

func (d SQLiteDialect) GetPrimaryKeyType() string {
	return "INTEGER PRIMARY KEY AUTOINCREMENT"
}

func (d SQLiteDialect) GetBlobType() string {
	return "BLOB"
}

func (d SQLiteDialect) Initialize(db *sql.DB) error {
	// WALモードを有効化
	if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		return fmt.Errorf("failed to enable WAL mode: %w", err)
	}
	return nil
}

func (d SQLiteDialect) ConvertPlaceholders(query string) string {
	// $1, $2, ... を ? に変換
	re := regexp.MustCompile(`\$(\d+)`)
	return re.ReplaceAllString(query, "?")
}

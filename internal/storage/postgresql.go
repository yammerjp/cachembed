package storage

import (
	"database/sql"

	_ "github.com/lib/pq"
)

// PostgreSQLDialect はPostgreSQL用の実装
type PostgreSQLDialect struct{}

func (d PostgreSQLDialect) GetPrimaryKeyType() string {
	return "BIGSERIAL PRIMARY KEY"
}

func (d PostgreSQLDialect) GetBlobType() string {
	return "BYTEA"
}

func (d PostgreSQLDialect) Initialize(db *sql.DB) error {
	// PostgreSQLでは特別な初期化は不要
	return nil
}

func (d PostgreSQLDialect) ConvertPlaceholders(query string) string {
	// PostgreSQLではそのまま返す
	return query
}

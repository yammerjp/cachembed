package storage

import (
	"fmt"
	"net/url"
	"strings"
)

type dbConfig struct {
	Driver  string
	DSN     string
	Dialect Dialect
}

func parseDSN(dsn string) (*dbConfig, error) {
	if strings.HasSuffix(dsn, ".db") || strings.HasPrefix(dsn, "file:") || strings.HasPrefix(dsn, ":memory:") {
		return &dbConfig{
			Driver:  "sqlite3",
			DSN:     dsn,
			Dialect: SQLiteDialect{},
		}, nil
	}

	u, err := url.Parse(dsn)
	if err != nil {
		return nil, fmt.Errorf("invalid DSN format: %w", err)
	}

	switch u.Scheme {
	case "postgres", "postgresql":
		return &dbConfig{
			Driver:  "postgres",
			DSN:     dsn,
			Dialect: PostgreSQLDialect{},
		}, nil
	default:
		return nil, fmt.Errorf("unsupported database type: %s (only sqlite3 and postgres are supported)", u.Scheme)
	}
}

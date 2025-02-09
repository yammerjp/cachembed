package storage

import "database/sql"

type Dialect interface {
	GetPrimaryKeyType() string
	GetBlobType() string
	Initialize(db *sql.DB) error
	ConvertPlaceholders(query string) string
}

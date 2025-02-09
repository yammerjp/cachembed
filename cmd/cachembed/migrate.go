package cachembed

import (
	"database/sql"
	"log/slog"
	"os"

	"github.com/yammerjp/cachembed/internal/storage"
)

func runMigration(dsn string) {
	slog.Info("running database migration", "dsn", dsn)

	config, err := storage.ParseDSN(dsn)
	if err != nil {
		slog.Error("failed to parse DSN", "error", err)
		os.Exit(1)
	}

	db, err := sql.Open(config.Driver, config.DSN)
	if err != nil {
		slog.Error("failed to open database", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		slog.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}

	// マイグレーションの実行（Dialectを渡す）
	if err := storage.RunMigrations(db, config.Dialect); err != nil {
		slog.Error("failed to run migrations", "error", err)
		os.Exit(1)
	}

	slog.Info("database migration completed successfully")
}

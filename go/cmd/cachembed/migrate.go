package cachembed

import (
	"log/slog"
	"os"

	"github.com/yammerjp/cachembed/internal/storage"
)

func runMigration(dsn string) {
	slog.Info("running database migration", "dsn", dsn)

	db, err := storage.NewDB(dsn)
	if err != nil {
		slog.Error("failed to open database", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		slog.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}

	if err := db.RunMigrations(); err != nil {
		slog.Error("failed to run migrations", "error", err)
		os.Exit(1)
	}

	slog.Info("database migration completed successfully")
}

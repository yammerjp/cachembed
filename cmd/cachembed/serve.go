package cachembed

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"

	"github.com/yammerjp/cachembed/internal/handler"
	"github.com/yammerjp/cachembed/internal/storage"
)

func runServer(cmd ServeCmd, dsn string, debugBody bool) {
	slog.Info("starting server",
		"host", cmd.Host,
		"port", cmd.Port,
		"upstream_url", cmd.UpstreamURL,
		"allowed_models", cmd.AllowedModels,
	)

	// データベースの初期化
	db, err := storage.NewDB(dsn)
	if err != nil {
		slog.Error("failed to initialize database", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	// ハンドラの作成
	handler := handler.NewHandler(cmd.AllowedModels, cmd.APIKeyPattern, cmd.UpstreamURL, db, debugBody)

	// サーバーの起動
	addr := fmt.Sprintf("%s:%d", cmd.Host, cmd.Port)
	server := &http.Server{
		Addr:    addr,
		Handler: handler,
	}

	slog.Info("server is ready", "addr", addr)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		slog.Error("server error", "error", err)
		os.Exit(1)
	}
}

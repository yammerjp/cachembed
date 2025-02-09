package main

import (
	"database/sql"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/alecthomas/kong"
)

type CLI struct {
	Serve    ServeCmd   `cmd:"" help:"Start the Cachembed server."`
	GC       GCCmd      `cmd:"" help:"Manually trigger garbage collection for LRU cache."`
	Migrate  MigrateCmd `cmd:"" help:"Run database migrations."`
	LogLevel string     `help:"Logging level (debug, info, warn, error)." default:"info"`
	DSN      string     `help:"Database connection string. Use file path for SQLite (e.g., 'cache.db') or URL for PostgreSQL (e.g., 'postgres://user:pass@localhost/dbname')." default:"cachembed.db"`
}

type ServeCmd struct {
	Host          string   `help:"Host to bind the server." default:"127.0.0.1"`
	Port          int      `help:"Port to run the server on." default:"8080"`
	UpstreamURL   string   `help:"URL of the upstream embedding API." env:"CACHEMBED_UPSTREAM_URL" default:"https://api.openai.com/v1/embeddings"`
	AllowedModels []string `help:"List of allowed embedding models." env:"CACHEMBED_ALLOWED_MODELS" default:"text-embedding-3-small,text-embedding-3-large,text-embedding-ada-002"`
	APIKeyPattern string   `help:"Regular expression pattern for API key validation." env:"CACHEMBED_API_KEY_PATTERN" default:"^sk-[a-zA-Z0-9]+$"`
}

type GCCmd struct {
	Before  string `help:"Delete entries older than this duration (e.g., '24h', '7d')" required:""`
	StartID int64  `help:"Start ID for deletion (optional)"`
	EndID   int64  `help:"End ID for deletion (optional)"`
	Sleep   int    `help:"Sleep duration between iterations in seconds (optional)"`
}

type MigrateCmd struct{}

func setupLogger(levelStr string) {
	var level slog.Level
	switch levelStr {
	case "debug":
		level = slog.LevelDebug
	case "info":
		level = slog.LevelInfo
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: level,
	}))
	slog.SetDefault(logger)
}

func main() {
	var cli CLI
	ctx := kong.Parse(&cli,
		kong.Name("cachembed"),
		kong.Description("Lightweight caching proxy for OpenAI embedding API."),
		kong.UsageOnError(),
	)

	setupLogger(cli.LogLevel)

	switch ctx.Command() {
	case "serve":
		runServer(cli.Serve, cli.DSN)
	case "gc":
		runGarbageCollection(cli.GC, cli.DSN)
	case "migrate":
		runMigration(cli.DSN)
	default:
		log.Fatalf("Unknown command: %s", ctx.Command())
	}
}

func runServer(cmd ServeCmd, dsn string) {
	slog.Info("starting server",
		"host", cmd.Host,
		"port", cmd.Port,
		"upstream_url", cmd.UpstreamURL,
		"allowed_models", cmd.AllowedModels,
	)

	// データベースの初期化
	db, err := NewDB(dsn)
	if err != nil {
		slog.Error("failed to initialize database", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	// ハンドラの作成
	handler := newHandler(cmd.AllowedModels, cmd.APIKeyPattern, cmd.UpstreamURL, db)

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

func runGarbageCollection(cmd GCCmd, dsn string) {
	duration, err := parseDuration(cmd.Before)
	if err != nil {
		slog.Error("invalid duration format", "error", err, "value", cmd.Before)
		os.Exit(1)
	}

	db, err := NewDB(dsn)
	if err != nil {
		slog.Error("failed to initialize database", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	// 最大IDを取得（EndIDが指定されていない場合に使用）
	var maxID int64
	err = db.QueryRow("SELECT COALESCE(MAX(id), 0) FROM embeddings").Scan(&maxID)
	if err != nil {
		slog.Error("failed to get max ID", "error", err)
		os.Exit(1)
	}

	// EndIDが0（未指定）の場合は最大IDを使用
	endID := cmd.EndID
	if endID == 0 {
		endID = maxID
	}

	// GC実行
	if err := db.DeleteEntriesBeforeWithSleep(duration, cmd.StartID, endID, time.Duration(cmd.Sleep)*time.Second); err != nil {
		slog.Error("failed to run garbage collection", "error", err)
		os.Exit(1)
	}

	slog.Info("garbage collection completed successfully")
}

func runMigration(dsn string) {
	slog.Info("running database migration", "dsn", dsn)

	config, err := ParseDSN(dsn)
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
	if err := runMigrations(db, config.Dialect); err != nil {
		slog.Error("failed to run migrations", "error", err)
		os.Exit(1)
	}

	slog.Info("database migration completed successfully")
}

// parseDuration は "24h", "7d", "30d" のような文字列をtime.Durationに変換します
func parseDuration(s string) (time.Duration, error) {
	// 日単位の指定をサポート
	if strings.HasSuffix(s, "d") {
		days, err := strconv.Atoi(strings.TrimSuffix(s, "d"))
		if err != nil {
			return 0, fmt.Errorf("invalid day format: %w", err)
		}
		return time.Duration(days) * 24 * time.Hour, nil
	}

	// 時間単位の指定
	return time.ParseDuration(s)
}

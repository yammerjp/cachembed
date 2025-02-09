package cachembed

import (
	"log"
	"log/slog"
	"os"

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

func Main() {
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

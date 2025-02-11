package cachembed

import (
	"log"
	"log/slog"
	"os"

	"github.com/alecthomas/kong"
)

type BuildInfo struct {
	Version string
	Commit  string
	Date    string
	BuiltBy string
}

var buildInfo BuildInfo

type CLI struct {
	Serve           ServeCmd           `cmd:"" help:"Start the Cachembed server."`
	GC              GCCmd              `cmd:"" help:"Manually trigger garbage collection for LRU cache."`
	Migrate         MigrateCmd         `cmd:"" help:"Run database migrations."`
	MigrateAndServe MigrateAndServeCmd `cmd:"" help:"Run database migrations and start the server."`
	Version         VersionCmd         `cmd:"" help:"Show version information."`
	LogLevel        string             `help:"Logging level (debug, info, warn, error)." env:"CACHEMBED_LOG_LEVEL" default:"info"`
	DSN             string             `help:"Database connection string. Use file path for SQLite (e.g., 'cache.db') or URL for PostgreSQL (e.g., 'postgres://user:pass@localhost/dbname')." env:"CACHEMBED_DSN" default:"cachembed.db"`
}

type ServeCmd struct {
	Host          string   `help:"Host to bind the server." env:"CACHEMBED_HOST" default:"127.0.0.1"`
	Port          int      `help:"Port to run the server on." env:"CACHEMBED_PORT" default:"8080"`
	UpstreamURL   string   `help:"URL of the upstream embedding API." env:"CACHEMBED_UPSTREAM_URL" default:"https://api.openai.com/v1/embeddings"`
	AllowedModels []string `help:"List of allowed embedding models." env:"CACHEMBED_ALLOWED_MODELS" default:"text-embedding-3-small,text-embedding-3-large,text-embedding-ada-002"`
	APIKeyPattern string   `help:"Regular expression pattern for API key validation." env:"CACHEMBED_API_KEY_PATTERN" default:"^sk-[a-zA-Z0-9_-]+$"`
	DebugBody     bool     `help:"Debug request body." env:"CACHEMBED_DEBUG_BODY" default:"false"`
}

type GCCmd struct {
	Before  string `help:"Delete entries older than this duration (e.g., '24h', '7d')" required:""`
	StartID int64  `help:"Start ID for deletion (optional)"`
	EndID   int64  `help:"End ID for deletion (optional)"`
	Batch   int    `help:"Batch size for deletion (optional)" default:"1000"`
	Sleep   int    `help:"Sleep duration between iterations in seconds (optional)"`
}

type MigrateCmd struct{}

type MigrateAndServeCmd struct {
	ServeCmd
}

type VersionCmd struct{}

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

func Run(bi BuildInfo) {
	buildInfo = bi

	var cli CLI
	ctx := kong.Parse(&cli,
		kong.Name("cachembed"),
		kong.Description("Lightweight caching proxy for OpenAI embedding API."),
		kong.UsageOnError(),
	)

	setupLogger(cli.LogLevel)

	switch ctx.Command() {
	case "serve":
		runServer(cli.Serve, cli.DSN, cli.Serve.DebugBody)
	case "gc":
		runGarbageCollection(cli.GC, cli.DSN)
	case "migrate":
		runMigration(cli.DSN)
	case "migrate-and-serve":
		runMigration(cli.DSN)
		runServer(cli.MigrateAndServe.ServeCmd, cli.DSN, cli.MigrateAndServe.DebugBody)
	case "version":
		runVersion()
	default:
		log.Fatalf("Unknown command: %s", ctx.Command())
	}
}

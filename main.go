package main

import (
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"

	"github.com/alecthomas/kong"
)

type CLI struct {
	Serve    ServeCmd `cmd:"" help:"Start the Cachembed server."`
	GC       GCCmd    `cmd:"" help:"Manually trigger garbage collection for LRU cache."`
	LogLevel string   `help:"Logging level (debug, info, warn, error)." default:"info"`
}

type ServeCmd struct {
	DSN           string   `help:"Path to the SQLite database file." default:"cachembed.db"`
	Host          string   `help:"Host to bind the server." default:"127.0.0.1"`
	Port          int      `help:"Port to run the server on." default:"8080"`
	UpstreamURL   string   `help:"URL of the upstream embedding API." env:"CACHEMBED_UPSTREAM_URL" default:"https://api.openai.com/v1/embeddings"`
	AllowedModels []string `help:"List of allowed embedding models." env:"CACHEMBED_ALLOWED_MODELS" default:"text-embedding-3-small,text-embedding-3-large,text-embedding-ada-002"`
	APIKeyPattern string   `help:"Regular expression pattern for API key validation." env:"CACHEMBED_API_KEY_PATTERN" default:"^sk-[a-zA-Z0-9]+$"`
}

type GCCmd struct {
	GCLimit int `help:"Number of least recently used items to remove." default:"100"`
}

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
		startServer(cli.Serve)
	case "gc":
		runGarbageCollection(cli.GC)
	default:
		log.Fatalf("Unknown command: %s", ctx.Command())
	}
}

func startServer(cmd ServeCmd) {
	slog.Info("starting server",
		"host", cmd.Host,
		"port", cmd.Port,
		"database", cmd.DSN,
		"upstream_api", cmd.UpstreamURL,
		"allowed_models", cmd.AllowedModels,
	)

	db, err := NewDB(cmd.DSN)
	if err != nil {
		slog.Error("failed to initialize database", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	handler := newHandler(cmd.AllowedModels, cmd.APIKeyPattern, cmd.UpstreamURL)

	addr := fmt.Sprintf("%s:%d", cmd.Host, cmd.Port)
	if err := http.ListenAndServe(addr, handler); err != nil {
		slog.Error("server failed to start", "error", err)
		os.Exit(1)
	}
}

func runGarbageCollection(cmd GCCmd) {
	slog.Info("running garbage collection", "gc_limit", cmd.GCLimit)
	// TODO: Implement GC logic
	os.Exit(1)
}

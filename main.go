package main

import (
	"fmt"
	"log"
	"log/slog"
	"os"

	"github.com/alecthomas/kong"
)

type CLI struct {
	Serve    ServeCmd `cmd:"" help:"Start the Cachembed server."`
	GC       GCCmd    `cmd:"" help:"Manually trigger garbage collection for LRU cache."`
	LogLevel string   `help:"Logging level (debug, info, warn, error)." default:"info"`
}

type ServeCmd struct {
	DSN         string `help:"Path to the SQLite database file." default:"cachembed.db"`
	Host        string `help:"Host to bind the server." default:"127.0.0.1"`
	Port        int    `help:"Port to run the server on." default:"8080"`
	UpstreamURL string `help:"URL of the upstream embedding API." env:"CACHEMBED_UPSTREAM_URL" default:"https://api.openai.com/v1/embeddings"`
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
	fmt.Printf("Starting server on %s:%d using database: %s\n", cmd.Host, cmd.Port, cmd.DSN)
	fmt.Printf("Upstream API: %s\n", cmd.UpstreamURL)
	os.Exit(1)
}

func runGarbageCollection(cmd GCCmd) {
	fmt.Printf("Running garbage collection, removing %d least recently used items\n", cmd.GCLimit)
	// TODO: Implement GC logic
	os.Exit(1)
}

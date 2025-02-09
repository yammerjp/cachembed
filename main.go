package main

import (
	"fmt"
	"log"
	"os"

	"github.com/alecthomas/kong"
)

type CLI struct {
	Serve ServeCmd `cmd:"" help:"Start the Cachembed server."`
	GC    GCCmd   `cmd:"" help:"Manually trigger garbage collection for LRU cache."`
}

type ServeCmd struct {
	DSN         string `help:"Path to the SQLite database file." default:"cachembed.db"`
	Host        string `help:"Host to bind the server." default:"127.0.0.1"`
	Port        int    `help:"Port to run the server on." default:"8080"`
	LogLevel    string `help:"Logging level (debug, info, warn, error)." default:"info"`
	UpstreamURL string `help:"URL of the upstream embedding API." env:"CACHEMBED_UPSTREAM_URL" default:"https://api.openai.com/v1/embeddings"`
}

type GCCmd struct {
	GCLimit int `help:"Number of least recently used items to remove." default:"100"`
}

func main() {
	var cli CLI
	ctx := kong.Parse(&cli,
		kong.Name("cachembed"),
		kong.Description("Lightweight caching proxy for OpenAI embedding API."),
		kong.UsageOnError(),
	)

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
	fmt.Printf("Upstream API: %s (log level: %s)\n", cmd.UpstreamURL, cmd.LogLevel)
	// TODO: Implement proxy server logic
  os.Exit(1)
}

func runGarbageCollection(cmd GCCmd) {
	fmt.Printf("Running garbage collection, removing %d least recently used items\n", cmd.GCLimit)
	// TODO: Implement GC logic
  os.Exit(1)
}

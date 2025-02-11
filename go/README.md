# Cachembed

A lightweight caching proxy for OpenAI embedding API requests.

## Overview

Cachembed is a proxy server that caches OpenAI embedding API results to reduce redundant requests and minimize costs. It supports SQLite (default) and PostgreSQL as storage backends, allows model restrictions, and provides garbage collection (GC) for cache management.

## Features

- Caches embedding results to SQLite or PostgreSQL.
- Proxies requests to OpenAI API (https://api.openai.com/v1/embeddings by default).
- Supports API key validation via regex pattern.
- Restricts usage to allowed embedding models.
- Provides garbage collection (GC) for cache cleanup.
- Supports database migrations.
- Configurable via command-line arguments or environment variables.

## Installation

```bash
go install github.com/yammerjp/cachembed@latest
```

## Usage

Cachembed provides the following commands:

### 1. Start the Server

#### Basic Usage

```sh
cachembed serve [options]
```

#### With Automatic Database Migration

```sh
cachembed migrate-and-serve [options]
```

#### Options

| Flag | Environment Variable | Description | Default |
|------|----------------------|-------------|---------|
| --host | CACHEMBED_HOST | Host to bind the server | 127.0.0.1 |
| --port | CACHEMBED_PORT | Port to run the server on | 8080 |
| --upstream-url | CACHEMBED_UPSTREAM_URL | OpenAI embedding API endpoint | https://api.openai.com/v1/embeddings |
| --allowed-models | CACHEMBED_ALLOWED_MODELS | Comma-separated list of allowed models | text-embedding-3-small,text-embedding-3-large,text-embedding-ada-002 |
| --api-key-pattern | CACHEMBED_API_KEY_PATTERN | Regular expression pattern for API key validation | ^sk-[a-zA-Z0-9]+$ |
| --dsn | CACHEMBED_DSN | Database connection string (SQLite file path or PostgreSQL URL) | cachembed.db |
| --log-level | CACHEMBED_LOG_LEVEL | Logging level (debug, info, warn, error) | info |

#### Example

```bash
cachembed serve --host 0.0.0.0 --port 9090 --dsn "postgres://user:pass@localhost/cachedb"
```

### 2. Run Garbage Collection (GC)

```bash
cachembed gc [options]
```

#### Options

| Flag | Description | Required | Default |
|------|-------------|----------|---------|
| --before | Delete entries older than this duration (e.g., 24h, 7d) | ✅ | - |
| --start-id | Start ID for deletion (optional) | ❌ | - |
| --end-id | End ID for deletion (optional) | ❌ | - |
| --batch | Batch size for deletion | ❌ | 1000 |
| --sleep | Sleep duration between iterations in seconds | ❌ | - |

#### Example

```bash
cachembed gc --before "7d"
```

### 3. Run Database Migrations

```bash
cachembed migrate
```

Ensures that the database schema is up to date.

#### Example

```bash
cachembed migrate
```

### 4. Show Version

```bash
cachembed version
```

## Configuration via Environment Variables

Cachembed supports configuration via environment variables:

```bash
export CACHEMBED_DSN="cachembed.db"
export CACHEMBED_LOG_LEVEL="debug"
export CACHEMBED_UPSTREAM_URL="https://custom-api.com/v1/embeddings"
export CACHEMBED_ALLOWED_MODELS="text-embedding-3-small,text-embedding-3-large"
export CACHEMBED_API_KEY_PATTERN="^sk-[a-zA-Z0-9]+$"
```

## Docker

docker image is available at [ghcr.io/yammerjp/cachembed](https://github.com/orgs/yammerjp/packages/container/package/cachembed)

```bash
docker run -d -v cachembed.db:/cachembed.db -p 8080:8080 ghcr.io/yammerjp/cachembed:latest-amd64
# or
docker run -d -v cachembed.db:/cachembed.db -p 8080:8080 ghcr.io/yammerjp/cachembed:latest-arm64
```

## License

MIT License

## Contributing

Pull requests are welcome! If you find a bug or want to request a feature, please open an issue.

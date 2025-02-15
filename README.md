# Cachembed

A lightweight caching proxy for OpenAI embedding API requests (Rails implementation)

## Overview

Cachembed is a proxy server that caches OpenAI embedding API results to reduce redundant requests and minimize costs. It supports SQLite (default) and PostgreSQL as storage backends, allows model restrictions, and provides garbage collection (GC) for cache management.

## Features

- Caches embedding results to SQLite or PostgreSQL
- Proxies requests to OpenAI API (https://api.openai.com/v1/embeddings by default)
- Supports API key validation via regex pattern
- Restricts usage to allowed embedding models
- Supports database migrations
- Configurable via environment variables

## Requirements

* Ruby 3.4.1 or higher
* Rails 8.0.1 or higher
* SQLite3 or PostgreSQL

## Installation

Clone the repository and install dependencies:

    git clone https://github.com/your-username/cachembed-rails
    cd cachembed-rails
    # if you want to use PostgreSQL, run:
    bundle install --with=postgresql
    # if you want to use SQLite, run:
    bundle install

## Setup

1. Create and migrate the database:

    bin/setup --skip=server

2. Set up environment variables:

    cp .env.example .env
    # Edit .env file with your configuration

## Configuration

Configure the application using these environment variables:

| Environment Variable | Description | Default |
|---------------------|-------------|----------|
| CACHEMBED_UPSTREAM_URL | OpenAI embedding API endpoint | https://api.openai.com/v1/embeddings |
| CACHEMBED_ALLOWED_MODELS | Comma-separated list of allowed models | text-embedding-3-small,text-embedding-3-large,text-embedding-ada-002 |
| CACHEMBED_API_KEY_PATTERN | Regular expression pattern for API key validation | ^sk-[a-zA-Z0-9]+$ |
| DATABASE_URL | Database connection string | Depends on config/database.yml |

## Usage

### Starting the Server

Development environment:

    rails server

Production environment:

    RAILS_ENV=production rails server

### API Endpoints

The server provides the following endpoint:

- POST `/embeddings`: Proxies requests to OpenAI's embedding API with caching

Example request:

    curl -X POST http://localhost:3000/embeddings \
      -H "Content-Type: application/json" \
      -H "Authorization: Bearer sk-your-api-key" \
      -d '{
        "input": "Your text here",
        "model": "text-embedding-3-small"
      }'

## Docker

Run using Docker Compose:

    docker compose up -d

## License

MIT License

## Contributing

Pull requests are welcome! If you find a bug or want to request a feature, please open an issue.

## TODO

- LRU cache (with request logs)
- Garbage collection for old cache entries

# Cachembed

Ein leichtgewichtiger Caching-Proxy für OpenAI Embedding-API-Anfragen.

## Übersicht

Cachembed ist ein Proxy-Server, der die Ergebnisse der OpenAI Embedding-API zwischenspeichert, um redundante Anfragen zu reduzieren und die Kosten zu minimieren. Er unterstützt SQLite (Standard) und PostgreSQL als Speicher-Backends.

## Funktionen

- Zwischenspeicherung von Embedding-Ergebnissen in SQLite oder PostgreSQL
- Proxy für Anfragen an die OpenAI-API (standardmäßig https://api.openai.com/v1/embeddings)
- Unterstützt die Validierung des API-Schlüssels über ein Regex-Muster
- Beschränkt die Nutzung auf erlaubte Embedding-Modelle
- Unterstützt Datenbankmigrationen
- Konfigurierbar über Umgebungsvariablen

## Anforderungen

* Ruby 3.4.1 oder höher
* Rails 8.0.1 oder höher
* SQLite3 oder PostgreSQL

## Installation

Klonen Sie das Repository und installieren Sie die Abhängigkeiten:

    git clone https://github.com/your-username/cachembed-rails
    cd cachembed-rails
    # wenn Sie PostgreSQL verwenden möchten, führen Sie aus:
    bundle install --with=postgresql
    # wenn Sie SQLite verwenden möchten, führen Sie aus:
    bundle install

## Einrichtung

Erstellen und migrieren Sie die Datenbank:

    bin/setup --skip=server

## Konfiguration

Konfigurieren Sie die Anwendung mit diesen Umgebungsvariablen:

| Umgebungsvariable           | Beschreibung                                        | Standard                          |
|-----------------------------|----------------------------------------------------|-----------------------------------|
| CACHEMBED_UPSTREAM_URL      | OpenAI Embedding-API-Endpunkt                      | https://api.openai.com/v1/embeddings |
| CACHEMBED_ALLOWED_MODELS    | Komma-getrennte Liste von erlaubten Modellen       | text-embedding-3-small,text-embedding-3-large,text-embedding-ada-002 |
| CACHEMBED_API_KEY_PATTERN   | Reguläres Ausdruckmuster zur Validierung des API-Schlüssels | ^sk-[a-zA-Z0-9_-]+$             |
| DATABASE_URL                | Datenbankverbindungszeichenfolge                    | Hängt von config/database.yml ab  |

## Nutzung

### Server starten

Entwicklungsumgebung:

    rails server

Produktionsumgebung:

    RAILS_ENV=production rails server

### API-Endpunkte

Der Server bietet den folgenden Endpunkt:

- POST `/v1/embeddings`: Proxy für Anfragen an die Embedding-API von OpenAI mit Caching

Beispielanfrage:

    curl -X POST http://localhost:3000/v1/embeddings \
      -H "Content-Type: application/json" \
      -H "Authorization: Bearer sk-your-api-key" \
      -d '{
        "input": "Ihr Text hier",
        "model": "text-embedding-3-small"
      }'

## Lizenz

MIT-Lizenz

## Mitwirken

Pull-Requests sind willkommen! Wenn Sie einen Fehler finden oder eine Funktion anfordern möchten, öffnen Sie bitte ein Issue.

## TODO

- LRU-Cache (mit Anfrageprotokollen) und Garbage Collection für alte Cache-Einträge
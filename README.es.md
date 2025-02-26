# Cachembed

Un proxy de caché ligero para las solicitudes de la API de embeddings de OpenAI.

## Visión General

Cachembed es un servidor proxy que almacena en caché los resultados de la API de embeddings de OpenAI para reducir solicitudes redundantes y minimizar costos. Soporta SQLite (por defecto) y PostgreSQL como backends de almacenamiento.

## Características

- Almacena en caché resultados de embeddings en SQLite o PostgreSQL
- Proxies solicitudes a la API de OpenAI (https://api.openai.com/v1/embeddings por defecto)
- Soporta validación de clave de API a través de patrones regex
- Restringe el uso a modelos de embeddings permitidos
- Soporta migraciones de base de datos
- Configurable a través de variables de entorno

## Requisitos

* Ruby 3.4.1 o superior
* Rails 8.0.1 o superior
* SQLite3 o PostgreSQL

## Instalación

Clona el repositorio e instala las dependencias:

    git clone https://github.com/your-username/cachembed-rails
    cd cachembed-rails
    # si quieres usar PostgreSQL, ejecuta:
    bundle install --with=postgresql
    # si quieres usar SQLite, ejecuta:
    bundle install

## Configuración

Crea y migra la base de datos:

    bin/setup --skip=server

## Configuración

Configura la aplicación utilizando estas variables de entorno:

| Variable de Entorno | Descripción | Predeterminado |
|---------------------|-------------|----------------|
| CACHEMBED_UPSTREAM_URL | Endpoint de la API de embeddings de OpenAI | https://api.openai.com/v1/embeddings |
| CACHEMBED_ALLOWED_MODELS | Lista de modelos permitidos separada por comas | text-embedding-3-small,text-embedding-3-large,text-embedding-ada-002 |
| CACHEMBED_API_KEY_PATTERN | Patrón de expresión regular para la validación de la clave de API | ^sk-[a-zA-Z0-9_-]+$ |
| DATABASE_URL | Cadena de conexión a la base de datos | Depende de config/database.yml |

## Uso

### Iniciando el Servidor

Entorno de desarrollo:

    rails server

Entorno de producción:

    RAILS_ENV=production rails server

### Endpoints de la API

El servidor proporciona el siguiente endpoint:

- POST `/v1/embeddings`: Proxies solicitudes a la API de embeddings de OpenAI con caché

Ejemplo de solicitud:

    curl -X POST http://localhost:3000/v1/embeddings \
      -H "Content-Type: application/json" \
      -H "Authorization: Bearer sk-your-api-key" \
      -d '{
        "input": "Tu texto aquí",
        "model": "text-embedding-3-small"
      }'

## Licencia

Licencia MIT

## Contribuciones

¡Se aceptan solicitudes de extracción! Si encuentras un error o quieres solicitar una característica, por favor abre un problema.

## TODO

- Caché LRU (con registros de solicitudes) y recolección de basura para entradas de caché antiguas
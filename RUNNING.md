# Running ShortLink

This file contains the detailed setup instructions for running and testing the URL shortener locally.

## Prerequisites

- Go 1.26.4 or newer
- Docker and Docker Compose
- `make` for the convenience commands

Docker is the easiest way to run the full stack because it starts PostgreSQL, runs migrations, and starts the app with the correct environment variables.

## Run with Docker Compose

Start the application:

```sh
make docker-up
```

This command:

1. Starts PostgreSQL 17.
2. Runs the SQL migrations from `migrations/`.
3. Builds and starts the Go API server.

The API will be available at:

```text
http://localhost:8080
```

Stop the application:

```sh
make docker-down
```

PostgreSQL data is stored in the `postgres_data` Docker volume, so encoded URLs survive application container restarts.

## Try the API

Encode a URL:

```sh
curl -s -X POST http://localhost:8080/encode \
  -H 'Content-Type: application/json' \
  -d '{"url":"https://codesubmit.io/library/react"}'
```

Example response:

```json
{
  "short_url": "http://localhost:8080/1"
}
```

Decode the short URL:

```sh
curl -s -X POST http://localhost:8080/decode \
  -H 'Content-Type: application/json' \
  -d '{"short_url":"http://localhost:8080/1"}'
```

Example response:

```json
{
  "url": "https://codesubmit.io/library/react"
}
```

## Run Locally Without Docker Compose

You can also run the Go app directly if you already have PostgreSQL available.

1. Create a database named `url_shortener`.

2. Apply the migration:

```sh
psql 'postgres://postgres:postgres@localhost:5432/url_shortener?sslmode=disable' \
  -f migrations/000001_create_urls.up.sql
```

3. Export the required environment variables:

```sh
export DATABASE_URL='postgres://postgres:postgres@localhost:5432/url_shortener?sslmode=disable'
export BASE_URL='http://localhost:8080'
export PORT='8080'
```

4. Start the server:

```sh
go run ./cmd/server
```

## Run Tests

Run all unit tests:

```sh
go test ./...
```

Run only unit tests for a package:

```sh
go test ./internal/service
go test ./internal/handler
go test ./internal/shortener
```

Run the integration test:

```sh
go test -tags=integration ./integration -count=1
```

The integration test uses Testcontainers to start PostgreSQL, so Docker must be running.

## Troubleshooting

If port `8080` is already in use, change the `PORT` and `BASE_URL` values in `docker-compose.yml` or in your local environment.

If port `5432` is already in use, stop the existing PostgreSQL process or change the exposed host port in `docker-compose.yml`.

If integration tests fail with Docker socket errors, make sure Docker Desktop or the Docker daemon is running and that your user can access it.

If the app starts but requests fail with database errors, confirm the migration has run and the `urls` table exists.

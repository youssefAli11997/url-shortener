# Running ShortLink

This file contains the detailed setup instructions for running and testing the URL shortener locally.

## Requirements

- Go 1.26.4+
- Docker
- Docker Compose
- Make (optional, but recommended)

Docker Compose is the recommended way to run the project locally. It provisions PostgreSQL, applies database migrations, and starts the API with the required configuration.

## Run with Docker Compose

Start the application:

```sh
make docker-up
```

This command:

1. Starts PostgreSQL 17.
2. Runs the SQL migrations from `migrations/`.
3. Builds and starts the Go API server.

The first startup may take longer because Docker images need to be pulled and the application image must be built.

The API will be available at:

```text
http://localhost:8080
```

Stop the application:

```sh
make docker-down
```

PostgreSQL data is stored in the `postgres_data` Docker volume. As a result, encoded URLs remain available even after the API container is restarted or recreated.

Production deployment:

https://short-link-production-615e.up.railway.app

## Try the API

### Verify the service is healthy:

Locally:
```sh
curl http://localhost:8080/healthz
```

Production:
```sh
curl https://short-link-production-615e.up.railway.app/healthz
```

Example response:

```json
{
  "status": "ok"
}
```

### Encode a URL:

Locally:
```sh
curl -s -X POST http://localhost:8080/encode \
  -H 'Content-Type: application/json' \
  -d '{"url":"https://codesubmit.io/library/react"}'
```

Production:
```sh
curl -s -X POST https://short-link-production-615e.up.railway.app/encode \
  -H 'Content-Type: application/json' \
  -d '{"url":"https://codesubmit.io/library/react"}'
```

Example response:

```json
{
  "short_url": "http://localhost:8080/1"
}
```

### Decode the short URL:

Locally:
```sh
curl -s -X POST http://localhost:8080/decode \
  -H 'Content-Type: application/json' \
  -d '{"short_url":"http://localhost:8080/1"}'
```

Production:
```sh
curl -s -X POST https://short-link-production-615e.up.railway.app/decode \
  -H 'Content-Type: application/json' \
  -d '{"short_url":"https://short-link-production-615e.up.railway.app/1"}'
```

Example response:

```json
{
  "url": "https://codesubmit.io/library/react"
}
```

## Convenience Commands

```sh
make docker-up          # start database, run migrations, start API
make docker-down        # stop containers
make test               # run unit tests
make test-integration   # run integration tests

## Run Locally Without Docker Compose

You can also run the Go app directly if you already have PostgreSQL available.

1. Create a database named `url_shortener`.

2. Apply all migrations.

Using golang-migrate:

```sh
migrate \
  -path migrations \
  -database 'postgres://postgres:postgres@localhost:5432/url_shortener?sslmode=disable' \
  up

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

Run unit tests:

```sh
go test ./...
```

Run integration tests:

```sh
go test -tags=integration ./integration -count=1
```

The integration tests use Testcontainers to provision an isolated PostgreSQL container. Docker must be running, but no database setup is required.

## Troubleshooting

If port `8080` is already in use, change the `PORT` and `BASE_URL` values in `docker-compose.yml` or in your local environment.

If port `5432` is already in use, stop the existing PostgreSQL process or change the exposed host port in `docker-compose.yml`.

If integration tests fail with Docker socket errors, make sure Docker Desktop or the Docker daemon is running and that your user can access it.

If the app starts but requests fail with database errors, confirm the migration has run and the `urls` table exists.

## Project Structure

```
cmd/server        Entry point
internal/app      Application wiring
internal/handler  HTTP handlers
internal/service  Business logic
internal/repository Database access
internal/model    Domain models
internal/middleware HTTP middleware
migrations        Database schema migrations
integration       End-to-end tests
```
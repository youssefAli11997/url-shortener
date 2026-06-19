# ShortLink

A small URL shortening service written in Go. It exposes two JSON endpoints:

- `POST /encode` stores an original URL and returns a shortened URL.
- `POST /decode` accepts a shortened URL and returns the original URL.

The service persists URLs in PostgreSQL, so previously encoded URLs can still be decoded after the application process restarts.

## Tech Stack

- Go standard `net/http` router
- PostgreSQL for persistent storage
- `pgx` for database access
- Docker Compose for local app, database, and migration orchestration
- Testcontainers for end-to-end integration testing

## Architecture

The application follows a layered design:

```
HTTP Handler
     ↓
Service Layer
     ↓
Repository Layer
     ↓
PostgreSQL
```

Responsibilities are separated to keep the codebase maintainable and easily testable. Dependencies are injected, allowing unit tests to replace repositories and services with mocks.

## API

### Encode

Request:

```http
POST /encode
Content-Type: application/json
```

```json
{
  "url": "https://codesubmit.io/library/react"
}
```

Successful response:

```json
{
  "short_url": "http://localhost:8080/1"
}
```

### Decode

Request:

```http
POST /decode
Content-Type: application/json
```

```json
{
  "short_url": "http://localhost:8080/1"
}
```

Successful response:

```json
{
  "url": "https://codesubmit.io/library/react"
}
```

Error responses are returned in JSON:

```json
{
  "error": "invalid url"
}
```

### Health Check

Request:

```http
GET /healthz
```

Response:

```json
{
  "status": "ok"
}
```

## How It Works

1. `/encode` validates the submitted URL.
2. The URL is inserted into PostgreSQL, or the existing row is reused if the same URL was already encoded.
3. The database row ID is encoded as a Base62 short code.
4. `/decode` extracts the short code, decodes it back to the database ID, and returns the stored original URL.

This design is deterministic and idempotent: encoding the same URL multiple times returns the same short URL and does not create duplicate records.

## Testing Strategy

The project contains multiple layers of testing:

- Unit tests for Base62 encoding and decoding.
- Unit tests for the service layer using mocked repositories.
- Unit tests for HTTP handlers using mocked services.
- End-to-end integration tests using Testcontainers and a real PostgreSQL instance.

Integration tests exercise the entire request flow and verify that data persists correctly in PostgreSQL.

## Running the Project

Detailed setup and run instructions are in [RUNNING.md](./RUNNING.md).

Quick start with Docker:

```sh
make docker-up
```

The API will be available at `http://localhost:8080`.

## Tests

Run unit tests:

```sh
go test ./...
```

Run integration tests:

```sh
go test -tags=integration ./integration -count=1
```

The test suite includes:

- Base62 encoder/decoder tests
- URL service unit tests
- HTTP handler unit tests for `/encode` and `/decode`
- Integration tests that start PostgreSQL with Testcontainers and round-trips `/encode` then `/decode`

The integration tests require Docker to be running.

## Configuration

The app reads configuration from environment variables:

| Variable | Example | Description |
| --- | --- | --- |
| `DATABASE_URL` | `postgres://postgres:postgres@localhost:5432/url_shortener?sslmode=disable` | PostgreSQL connection string |
| `BASE_URL` | `http://localhost:8080` | Base URL used when constructing short URLs |
| `PORT` | `8080` | HTTP port for the server |

## Attack Vectors

### 1. URL abuse / malicious input

Users may submit:

- extremely long URLs
- malformed URLs
- phishing links

Mitigation:

- URL validation (already implemented)
- optional max length limit


### 2. Denial of service

Mass URL submissions could overload DB.

Mitigation:

- rate limiting: production should limit requests by IP/API key and add protection against bursts. (not implemented but recommended)
- caching layer (Redis)

### 3. Database exhaustion

Unbounded growth of stored URLs.

Mitigation:

- add quotas, retention policies, and operational alerts.
- TTL-based cleanup
- archive strategy

### 4. SQL injection

Not applicable due to parameterized queries via pgx.

### 5. Predictable short URLs

The current implementation generates short URLs from sequential PostgreSQL IDs encoded in Base62. While this approach guarantees uniqueness and avoids collisions, it produces predictable short codes.

An attacker could enumerate short URLs and potentially discover or access URLs that were never intended to be publicly indexed. This could expose sensitive or private information contained in the original URLs.

Possible mitigations include:

- Generating random short codes from a sufficiently large keyspace and enforcing uniqueness with a database constraint.
- Introducing a distributed ID generator combined with an obfuscation layer instead of exposing sequential IDs directly.
- Implementing authentication and authorization for private links.
- Applying rate limiting to reduce large-scale enumeration attempts.
- Monitoring and alerting on suspicious access patterns.

The current implementation prioritizes simplicity and deterministic behavior, but a production-grade system containing sensitive or private URLs should avoid exposing sequential identifiers directly.

### 6. URL abuse and phishing

A public shortener can hide malicious destinations.

Mitigation:

- Production use should add abuse reporting, blocklists, malware scanning, and administrative takedown tools.

### 7. Transport security

local examples use HTTP.

Mitigation:

- Production should run behind HTTPS.

### 8. Error exposure

Implementation avoids exposing internal database or infrastructure details. (already implemented)

### 9. Server-side request forgery (SSRF)

The current implementation only stores URLs and does not fetch or inspect them, which eliminates SSRF risk in the current design. However, if future features such as link previews, metadata extraction, or screenshots are introduced, care must be taken to prevent requests to private networks, localhost, cloud metadata endpoints, and internal hostnames.


## Scalability Considerations

### Current Design

The service uses PostgreSQL as the source of truth. Each URL is stored once and assigned an auto-incrementing ID, which is encoded into a Base62 string to generate the short URL.

This approach has several advantages:

- Deterministic short URL generation.
- No collision handling is required.
- Database lookups are efficient through indexed primary keys.
- The application itself is stateless and can be horizontally scaled.

### Horizontal Scaling

Since the application maintains no local state, multiple instances can be deployed behind a load balancer. PostgreSQL remains the single source of truth and guarantees unique IDs through its sequence mechanism.

```
Clients
    ↓
Load Balancer
    ↓
Multiple API Instances
    ↓
PostgreSQL
```

### Read Scaling

In a production environment, most traffic is expected to be read-heavy (short URL → original URL).

To reduce database load, a caching layer such as Redis can be introduced:

```
Client
  ↓
API
  ↓
Redis Cache
  ↓ (cache miss)
PostgreSQL
```

Frequently accessed URLs could be served directly from cache, significantly reducing latency and database pressure.

### Write Scaling

At very large scale, PostgreSQL could become a bottleneck for write throughput.

Possible approaches include:

- Database partitioning or sharding.
- Decoupling ID generation from the database.
- Using asynchronous processing for non-critical workloads such as abuse scanning, expiry, analytics, and cleanup.

### Collision Handling

This implementation does not require collision detection because short URLs are generated from unique PostgreSQL IDs encoded with Base62. As long as ID generation remains unique, collisions cannot occur.

### High Availability and ID Generation

The application itself is stateless and can be scaled horizontally behind a load balancer. However, in the current implementation PostgreSQL acts as both the source of truth and the ID generator through its sequence mechanism. This makes the database a single point of failure and a potential bottleneck for write throughput.

In a production environment, we could decouple ID generation from a single database instance in order to improve availability and scalability:

#### Alternative ID Generation Strategies

If preserving deterministic ID-based short codes, the system could adopt a distributed ID generation approach, such as:

- Snowflake-style IDs.
- Allocation of database sequence ranges to individual application instances.
- A dedicated ID allocation service.

These approaches remove the dependency on a single database sequence while preserving collision-free deterministic short codes.

Another option is to abandon deterministic IDs entirely and generate random short codes from a sufficiently large keyspace, such as UUIDs. In that model, uniqueness is enforced by a database unique constraint, and collisions are handled by retrying code generation until an unused value is found.

The choice between deterministic IDs and random codes depends on the requirements for throughput, simplicity, and operational complexity.

### Future Improvements

Potential enhancements include:

- Redis caching for frequently accessed URLs.
- Rate limiting to protect against abuse.
- Metrics and monitoring (Prometheus/Grafana).
- Analytics and click counting.
- URL expiration and cleanup policies.
- Custom aliases for short URLs.
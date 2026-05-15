# Technology Decisions

> Last updated: 2026-05-16
> Status: Active

This document keeps only the technology decision record for the project: what was chosen, why it fits, and the main trade-offs.

---

## Decision Summary

| Area | Chosen technology | Main reason |
|---|---|---|
| Language | Go | Strong fit for HTTP services, Kafka consumers, and concurrent CSV import jobs |
| HTTP framework | Gin | Lightweight routing, middleware, and request binding without hiding standard Go concepts |
| Database | PostgreSQL 16 | Relational data, transactions, joins, foreign keys, recursive queries, and permission checks |
| SQL layer | `sqlc` + `pgx` | Explicit SQL with generated, type-safe Go methods |
| Migrations | Goose | Ordered, reviewable schema changes that can run at service startup |
| Cache / token state | Redis 7 | Fast hot-path lookups and TTL-based expiry |
| Events | Apache Kafka | Durable, replayable user events between services |
| Token signing | JWT RS256 + JWKS | Public-key verification without sharing the private signing key |
| Concurrency | Goroutines + worker pool | Bounded parallelism for CSV import and Kafka consumers |
| Observability | Loki + Promtail | Centralized log aggregation across services |

---

## Go

Go fits this project because the workload is HTTP services, Kafka consumers, and concurrent CSV import jobs — areas where Go's goroutine model, static binary, and standard library shine.

| Need | How Go helps |
|---|---|
| HTTP services | `net/http` plus Gin gives a clear, low-abstraction routing layer. |
| Concurrency | Goroutines and channels back the CSV worker pool and Kafka consumer groups. |
| Static binary | Single artifact per service; deploy and Docker images stay small. |
| Type system | Compile-time safety pairs well with `sqlc`-generated code. |
| Ecosystem | `pgx`, `gin-gonic/gin`, `segmentio/kafka-go`, and JWT libraries are mature. |

**Trade-off:** Generics are still limited and `if err != nil` is verbose, but the simplicity, fast builds, and ease of onboarding outweigh the boilerplate cost.

---

## Gin

Gin is used because it gives the project the HTTP features it needs without a large framework.

| Need | How Gin helps |
|---|---|
| Route grouping | Keeps `/v1`, public routes, and authenticated routes clear. |
| Middleware | JWT auth, request context, logging, and error handling fit naturally. |
| Request binding | Request DTOs can be decoded and validated near handlers. |
| Low abstraction | The code remains close to standard Go HTTP concepts. |

**Trade-off:** Gin adds a dependency over `net/http`, but the project gets cleaner routing and middleware structure with very little framework overhead.

---

## PostgreSQL 16

PostgreSQL is preferred because the domain is relational and permission-sensitive.

```text
core-service DB:
  users_projection ── team_members ── teams
  assets ──────────── asset_acl ───── users_projection
  assets.parent_id ──► assets          (self-ref: folder → child note)

auth-service DB:
  users (canonical) ── import_jobs
```

The project needs transactions, joins, foreign keys, `ON CONFLICT`, recursive queries, and predictable consistency for permission decisions. Those requirements fit PostgreSQL directly.

**Trade-off:** A relational schema requires migrations when the structure changes, but that cost is worth it for strong consistency and clear data relationships.

---

## sqlc + pgx

`sqlc` is used instead of an ORM because this project benefits from explicit SQL.

```text
db/query/*.sql
      │
      │ sqlc generate
      ▼
internal/db/*.go
      │
      │ repository calls generated methods
      ▼
domain service
```

| Choice | Benefit |
|---|---|
| Handwritten SQL | Queries are reviewable, optimizable, and close to the schema. |
| Generated Go methods | Compile-time type checking catches many query and scan mistakes. |
| No lazy-loading ORM behavior | Avoids hidden N+1 query patterns. |
| `pgx` driver | Gives idiomatic and efficient PostgreSQL access. |

**Trade-off:** Query changes require running `sqlc generate`, but the generated code keeps repository calls type-safe and predictable.

---

## Goose

Goose is used for schema migrations because migrations should be explicit, ordered, and reviewable.

| Why Goose | Impact |
|---|---|
| Versioned migration files | Schema history is visible in git. |
| Startup migration support | Local development is easier because services can prepare their own schema. |
| SQL-first workflow | Migration logic stays close to PostgreSQL behavior. |

**Trade-off:** Developers must write and maintain migrations, but this is safer than implicit auto-migration for a service boundary. Running migrations on startup keeps local dev simple; in production with multiple replicas, a startup lock or a separate migration job is needed to prevent concurrent migration runs.

---

## Kafka

Kafka is used for durable, replayable user events.

```text
auth-service              Kafka                         core-service
    │                       │                                │
    │ publish USER_CREATED  │                                │
    │ publish USER_UPDATED ►│ topic: user.events             │
    │ publish USER_DELETED  │                                │
    │                       │   consumer group:              │
    │                       │   core-user-projection ───────►│
    │                       │                                │ upsert / soft-delete
    │                       │                                │ users_projection (local table)
```

| Capability | Why it matters here |
|---|---|
| Persistent event log | `core-service` can recover missed user events after downtime. |
| Replay | A projection table can be rebuilt from historical events. |
| Consumer groups | Multiple consumers can share work without duplicate handling in the same group. |
| Production pattern | Event-driven projections are a core learning objective of the project. |

**Trade-off:** Kafka adds operational weight because it requires a broker and, in this local setup, Zookeeper. That complexity is accepted because the project is designed to teach event-driven service boundaries.

---

## Goroutines + worker pool (CSV import)

Bulk user import is CPU- and DB-bound, so auth-service uses a fixed-size goroutine pool instead of spawning one goroutine per row.

```text
CSV reader ─► jobs channel ─► worker 1 ┐
                              worker 2 ├─► repository + Kafka publish
                              worker N ┘
                              (N = IMPORT_WORKERS)
```

| Choice | Reason |
|---|---|
| Bounded pool (`IMPORT_WORKERS`) | Caps DB connections and Kafka producer pressure. |
| Channel-based job dispatch | Idiomatic Go back-pressure when workers are slow. |
| Errors aggregated, not fail-fast | A single bad row should not abort the whole import. |

**Trade-off:** A worker pool is a few extra lines compared to `go func(){}` per row, but it prevents the service from saturating PostgreSQL or Kafka when an import file is large.

---

## Redis 7

Redis is used where TTL and hot-path speed matter more than relational querying.

```text
┌──────────────────────────────┬─────────────────────────────┐
│ jwt:blacklist:{jti}           │ revoked access tokens       │
│ refresh token state           │ refresh rotation/revocation │
│ core cache keys               │ team / asset / ACL hot data │
└──────────────────────────────┴─────────────────────────────┘
```

Redis fits these workloads because the data is small, frequently checked, and naturally time-limited.

### Cache invalidation

- **TTL-based** for stable lookups (e.g. team membership cached for a few minutes) — accepts brief staleness in exchange for simple code.
- **Event-driven** for security-sensitive data: ACL or membership changes evict the key immediately so a removed user loses access on the next request.
- **Fail-open vs. fail-closed:** each Redis-dependent path declares its policy. Read-through caches fall back to PostgreSQL on Redis failure (fail-open); revocation checks rely on the short access-token TTL (≤15 min) as the safety net during a Redis outage.

**Trade-off:** Redis is another runtime dependency. Code paths must decide whether Redis errors fail closed or fail open depending on the security and availability requirement.

---

## JWT RS256 + JWKS

JWT access tokens are signed with RS256 so only `auth-service` needs the private key.

```text
auth-service
  │ keeps RSA private key
  │ signs JWT access token
  ▼
client sends token
  ▼
core-service
  │ fetches public key through JWKS
  │ verifies token signature
  │ never receives auth private key
```

RS256 is used instead of HS256 because verification services only need public keys. Adding another service later does not require distributing a shared signing secret.

JWKS also enables **key rotation without redeploying verification services**: the endpoint can publish multiple public keys identified by `kid`, auth-service starts signing with the new `kid`, and core-service picks up the new key on its next JWKS refresh. With a shared HMAC secret, rotation requires coordinated redeployment of every consumer.

**Trade-off:** RS256 requires key-pair management and JWKS caching, but it gives stronger service isolation and easier rotation than a shared HMAC secret.

---

## Loki + Promtail

Logs from both services are shipped to Loki via Promtail so they can be queried from a single place.

```text
auth-service ┐
             ├─► stdout (JSON) ─► Promtail (file scrape) ─► Loki ─► Grafana
core-service ┘
```

| Choice | Reason |
|---|---|
| Loki over ELK | Lightweight for a learning project; indexes labels, not full text. |
| Promtail file scrape | Containers log to stdout; Docker writes to log files; Promtail tails them. |
| Structured JSON logs | LogQL queries can filter on `level`, `service`, `request_id`, etc. |

**Trade-off:** Loki's label-based indexing is less flexible than full-text search, but it is much cheaper to run and matches the log volume produced by a 2-service system.

---

## Trade-Off Matrix

| Area | Chosen approach | Alternative | Trade-off |
|---|---|---|---|
| Service split | 2 services: auth + core | 1 monolith | More infrastructure, but cleaner ownership and private-key isolation |
| Service split | Team + asset together in core | Separate team and asset services | Fewer network calls and no distributed transaction for ACL/team checks |
| API style | REST over HTTP/JSON | gRPC | Easier to inspect and test with `curl`/Postman; less strict contract than protobuf |
| Persistence | PostgreSQL | MongoDB | Strong joins and transactions; schema changes require migrations |
| SQL layer | `sqlc` + `pgx` | GORM | Explicit, type-safe SQL; requires regeneration after query edits |
| Events | Kafka | RabbitMQ or Redis Pub/Sub | Durable replayable events; more local infrastructure |
| Token signing | RS256 + JWKS | HS256 shared secret | Better service isolation; requires key-pair management |
| Cache/token state | Redis | PostgreSQL-only | Built-in TTL and fast lookups; another runtime dependency |
| Concurrency | Worker pool with `IMPORT_WORKERS` | Goroutine per row | Bounded resource use; minor extra wiring |
| Migrations | Run on service startup via Goose | CI/CD pipeline step | Simpler local dev; multiple instances need a startup lock in production |
| Observability | Loki + Promtail | Elasticsearch + Kibana | Cheaper to run; label-based queries less flexible than full text |

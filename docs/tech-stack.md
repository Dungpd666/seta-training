# Tech Stack

## Overview

This project is a pair of Go microservices — **auth-service** and **core-service** — that together manage users, teams, and digital assets with role-based access control (RBAC) and event-driven data synchronization. The stack was chosen to reflect production-grade patterns while remaining learnable for a Go intern course capstone.

---

## Gin Framework

**Framework — Gin**: Gin was chosen over the standard `net/http` mux because it provides built-in request-binding, middleware chaining, and route grouping out of the box. Its performance characteristics are well-understood and it adds minimal abstraction over the standard library, keeping the codebase readable.

---

## PostgreSQL 16

Each service owns its own PostgreSQL database (`authdb` and `coredb`), enforcing the microservice principle of database isolation. PostgreSQL was preferred over alternatives for several reasons:

- **ACID guarantees**: user registration, token issuance, and asset ownership changes all require strong consistency — eventual consistency would introduce hard-to-debug edge cases.
- **Rich SQL feature set**: window functions, JSON operators, and `ON CONFLICT` clauses are used directly in queries, which would require workarounds in simpler stores.
- **Maturity and ecosystem**: excellent tooling (`psql`, `pgAdmin`, `pg_isready` health checks) and a well-documented wire protocol.

**Trade-off**: Running two separate PostgreSQL instances increases infrastructure cost compared to a shared database, but it guarantees that a migration or schema change in one service cannot break the other. The isolation boundary is worth the overhead for a system designed to scale independently.

**DB Access Layer — sqlc + pgx**: Raw SQL queries are annotated in `.sql` files and compiled by `sqlc` into type-safe Go methods. This was chosen over an ORM (e.g., GORM) because it keeps SQL explicit and reviewable, avoids N+1 query pitfalls introduced by lazy loading, and generates code that is easy to test.

**Migrations — Goose**: Schema migrations run automatically on startup using Goose's embedded migration runner instead of GORM auto-migration to avoid error.

---

## Apache Kafka

Kafka is used for **user event projection**: auth-service publishes `USER_CREATED`, `USER_UPDATED`, and `USER_DELETED` events to the `user.events` topic; core-service consumes them and maintains a local `users_projection` table.

This design avoids synchronous HTTP calls between services at query time. Without Kafka, every team membership check or asset ownership lookup in core-service would require a live HTTP call to auth-service, creating tight coupling and a single point of failure. With the projection pattern, core-service can validate team members entirely from its local database even when auth-service is temporarily unavailable.

**Why Kafka over alternatives (e.g., RabbitMQ)**:

- Kafka's persistent, replayable log means a restarted core-service consumer can replay missed events and reconstruct its projection — no data is lost during downtime.
- The consumer group mechanism (`core-user-projection`) allows horizontal scaling of consumers without duplicating events.
- The `segmentio/kafka-go` library provides a clean, idiomatic Go API without requiring CGO or a native Kafka client.

**Trade-off**: Kafka adds operational complexity — it requires Zookeeper and a broker process. For a small project, RabbitMQ may be more approriate. But, Kafka is the right choice here because event-driven architecture is a core learning objective and because the patterns it introduces apply directly to production systems.

---

## Redis 7

Redis serves two purposes: a **token blacklist** (`jwt:blacklist:{jti}`) for stateless JWT revocation, and a **refresh token store** for rotate-on-use invalidation and full session revocation (`RevokeAllForUser`). Redis was preferred over a PostgreSQL table because it natively supports TTL-based expiry — no cleanup jobs needed — and delivers sub-millisecond lookups on the hot path of every authenticated request.

---

## JWT RS256 + JWKS

Tokens are signed with RSA-2048 private keys (RS256) rather than a shared HMAC secret. This means auth-service's private key never leaves that service — core-service only needs the public key to verify signatures. The public key is served dynamically at `GET /.well-known/jwks.json`, following the JWKS standard. Core-service caches keys by `kid` header using `sync.RWMutex` and refreshes on unknown key ID.

**Why RS256 over HS256**: asymmetric signing eliminates the need to distribute a shared secret to every service that needs to verify tokens. Adding a third service in the future requires no secret rotation — it simply fetches the JWKS endpoint.

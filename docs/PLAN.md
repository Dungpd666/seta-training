# SETA Training Project — 1-Month Implementation Plan (Junior Level)

## Overview

This plan breaks the SETA project into 4 weeks of focused, manageable tasks. Each week ends with a working demo checkpoint. Tasks are sized for a junior Go developer — expect to spend time reading docs, debugging, and understanding patterns alongside writing code.

---

## Architecture: 2 Services

```
┌─────────────────────┐     ┌──────────────────────────────┐
│    auth-service     │     │      core-service            │
│    :8081            │     │      :8082                   │
│                     │     │                              │
│  - User register    │     │  - Team management           │
│  - Login / JWT      │     │  - Asset CRUD                │
│  - List users       │     │  - Asset sharing / RBAC      │
│  - Bulk CSV import  │     │  - Manager oversight         │
└─────────────────────┘     └──────────────────────────────┘
           │                              │
           └──────────────┬───────────────┘
                          │
              ┌───────────┼───────────┐
              │           │           │
         PostgreSQL      Kafka      Redis
```

**Why 2 services:**

- `auth-service` owns **identity** — who you are. Issues JWTs. Everything else just trusts the token.
- `core-service` owns **collaboration** — teams and assets are tightly coupled (manager oversight requires team membership, asset sharing depends on team roles). Keeping them together eliminates inter-service HTTP calls for every permission check.

Each service has its own PostgreSQL database — they never query each other's DB directly.

---

## Tech Stack

| Layer          | Choice                               | Reasoning                                                                           |
| -------------- | ------------------------------------ | ----------------------------------------------------------------------------------- |
| Language       | **Go 1.22+**                         | Required by course                                                                  |
| Router         | **Gin**                              | Most popular, most tutorials/examples online, easiest to Google errors              |
| Database       | **PostgreSQL**                       | Relational data (users, teams, ACL) fits SQL; joins make permission checks natural  |
| DB Layer       | **GORM (queries only)**              | Used for CRUD/queries. **Do not** use AutoMigrate — see Migrations                  |
| Migrations     | **`golang-migrate`**                 | Versioned SQL files in `migrations/`, run on boot. Auditable, rollbackable          |
| Auth           | **JWT RS256 (`golang-jwt/jwt/v5`)**  | Asymmetric: auth signs w/ private key, core verifies via JWKS. Core cannot forge    |
| Token lifetime | **Access 15m + refresh 7d (rotated)**| Short access TTL + revocable refresh → real logout, reuse-detection                 |
| Password       | **`bcrypt` (`golang.org/x/crypto`)** | Industry standard — never store plain or MD5 passwords                              |
| Message Broker | **Kafka (`segmentio/kafka-go`)**     | Industry standard, event replay, durable log — harder setup but strong resume value |
| Cache          | **Redis (`go-redis/v9`)**            | Industry standard, TTL built-in, required by docs                                   |
| Logging        | **`zerolog`**                        | Structured JSON logs, integrates cleanly with Loki/Promtail                         |
| Containers     | **Docker + docker-compose**          | Required by docs; multi-stage builds keep images small                              |

### Project structure (each service)

```
service-name/
├── cmd/
│   └── main.go
├── internal/
│   ├── handler/        # HTTP handlers
│   ├── service/        # business logic
│   ├── repository/     # database queries
│   └── model/          # structs
├── migrations/
├── Dockerfile
└── go.mod
```

---

## Week 1 — Auth Service (Days 1–7)

**Goal:** Users can register, log in, receive a JWT, and be imported via CSV.

### Day 1 — Project Setup

- [x] Create `auth-service/` with the structure above
- [x] `go mod init github.com/yourname/seta/auth-service`
- [x] Install: `gin`, `gorm`, `gorm/driver/postgres`, `golang-jwt`, `bcrypt`, `zerolog`, **`golang-migrate/migrate/v4`**, `go-redis/v9`, `google/uuid`
- [x] Write `docker-compose.yml` with PostgreSQL and Redis
- [x] Connect to DB and verify on startup with GET /health
- [x] Create `migrations/` directory with numbered `.up.sql` / `.down.sql` files. Run `migrate.Up()` on startup (fail fast if migration fails)

**Prompt for Claude Code:**

> "Set up a new Go microservice called auth-service. Use Gin router and GORM (for queries only — NOT AutoMigrate) with PostgreSQL. Create project structure: cmd/main.go, internal/handler, internal/service, internal/repository, internal/model, migrations/. Add GET /health endpoint. Use golang-migrate/migrate/v4 to apply versioned SQL migrations from migrations/ on startup. Use zerolog for structured logging."

### Day 2 — User Model

- [x] `model.User`: UserID UUID, Username, Email (unique), PasswordHash, Role (manager|member), CreatedAt
- [x] **Migration `000001_create_users.up.sql`** — `CREATE TABLE users (...)` with CHECK constraint on role, UNIQUE on email. Paired `.down.sql` drops the table.
- [x] **Migration `000002_create_refresh_tokens.up.sql`** — `refresh_tokens(jti uuid pk, user_id uuid, expires_at timestamp, revoked bool)`
- [x] `UserRepository`: `Create`, `FindByEmail`, `FindAll`
- [x] `RefreshTokenRepository`: `Insert`, `MarkRevoked`, `IsValid(jti)`

**Prompt for Claude Code:**

> "Write a GORM User model (UserID uuid pk, Username string, Email string unique, PasswordHash string, Role string 'manager'|'member', CreatedAt) — struct only, no AutoMigrate. Write SQL migration files in migrations/ for the users and refresh_tokens tables (include CHECK constraints and indexes). Write UserRepository with Create(user), FindByEmail(email), and FindAll() methods, plus RefreshTokenRepository with Insert, MarkRevoked, IsValid."

### Day 3 — Register, Login, Refresh, Logout (RS256)

- [ ] Generate an RS256 keypair at dev time: `openssl genpkey -algorithm RSA -pkeyopt rsa_keygen_bits:2048 -out jwt_rs256.pem && openssl rsa -in jwt_rs256.pem -pubout -out jwt_rs256.pub`. Mount both into the container via `JWT_PRIVATE_KEY_PATH` / `JWT_PUBLIC_KEY_PATH`.
- [x] `POST /register` — validate, hash password, save, return user (no password)
- [x] `POST /login` — verify bcrypt, return **`{access_token, refresh_token}`**
  - access: RS256, 15-min exp, claims `{sub, role, jti, iss, aud, exp}`
  - refresh: RS256, 7-day exp, `typ=refresh`; persist `jti` in `refresh_tokens`
- [x] `POST /refresh` — verify refresh token, check `refresh_tokens.revoked=false`, **rotate** (revoke old jti, issue new pair). If a revoked-or-unknown refresh is presented, revoke the entire chain for that user (token theft).
- [x] `POST /logout` — requires access token + refresh token in body:
  - mark refresh `jti` revoked in DB
  - push access-token `jti` into Redis with TTL = remaining exp (key: `jwt:blacklist:{jti}`)
  - return 204
- [x] `GET /.well-known/jwks.json` — returns the public key as a JWK set, so core-service can fetch it.

**Prompt for Claude Code:**

> "Implement auth-service JWT with RS256 using golang-jwt/jwt/v5. Load private key from JWT_PRIVATE_KEY_PATH, public key from JWT_PUBLIC_KEY_PATH. Endpoints: POST /register (role in manager|member, bcrypt, 201 or 409); POST /login returns {access_token (15m), refresh_token (7d)} with claims {sub, role, jti, iss=auth-service, aud=seta}; POST /refresh rotates the refresh token (revokes old jti in refresh_tokens and issues new pair — if an already-revoked refresh is presented, revoke all of that user's refresh tokens); POST /logout requires bearer + refresh body: revoke refresh jti in DB and SET jwt:blacklist:{access_jti} in Redis with TTL = remaining exp. Add GET /.well-known/jwks.json returning the public key as a JWKS."

### Day 4 — JWT Middleware & List Users

- [ ] Gin middleware: validate `Authorization: Bearer <token>` using the **public key** (RS256), verify `iss`, `aud`, `exp`, then check Redis `jwt:blacklist:{jti}` — 401 if present.
- [ ] Store parsed claims (sub → user_id, role) in Gin context.
- [ ] `GET /users` — protected, returns all users

**Prompt for Claude Code:**

> "Write a Gin JWT middleware that validates Authorization: Bearer <token> using golang-jwt with RS256 (public key from JWT_PUBLIC_KEY_PATH). Verify iss=auth-service, aud=seta, exp. Then check Redis key jwt:blacklist:{jti} — if present, return 401. Store claims (sub as user_id, role) in Gin context. Apply to GET /users which returns all users from DB (no password fields). Return 401 on any failure."

### Day 5–6 — Bulk CSV Import

- [ ] `POST /import-users` — multipart CSV file
- [ ] Worker pool: goroutines + channels, configurable pool size
- [ ] Returns `{succeeded, failed, errors: [{row, reason}]}`

**Prompt for Claude Code:**

> "Add POST /import-users to auth-service. Accept multipart/form-data CSV (columns: username, email, password, role). Process rows concurrently: jobs channel + 5 worker goroutines calling UserRepository.Create(). Collect results via results channel. Return {succeeded: int, failed: int, errors: [{row: int, reason: string}]}."

### Day 7 — Testing

- [ ] Manual test: register → login → get users → import CSV
- [ ] Unit tests for service layer

**Learning checkpoint:** Auth service fully working with JWT and concurrent CSV import.

---

## Week 2 — Core Service: Teams (Days 8–14)

**Goal:** Managers can create and manage teams.

### Day 8 — Setup, Team Models & Users Projection

- [ ] Create `core-service/` with same structure, including `migrations/`
- [ ] Models: `Team` (TeamID, TeamName, CreatedBy), `TeamMember` (TeamID, UserID, MemberRole), **`UserProjection` (UserID, Username, Email, Role, DeletedAt)**
- [ ] SQL migrations for all three tables. Add FKs: `teams.created_by → users_projection.user_id`, `team_members.user_id → users_projection.user_id`
- [ ] **Kafka consumer goroutine** on `user.events` (consumer group `core-user-projection`):
  - `USER_CREATED` / `USER_UPDATED` → upsert into `users_projection`
  - `USER_DELETED` → set `deleted_at = NOW()`
  - On startup, consumer starts from the earliest offset (so a fresh core-db backfills from the log)
- [ ] Use `golang-migrate`, **not** AutoMigrate.

**Prompt for Claude Code:**

> "Set up core-service (Gin + GORM for queries + PostgreSQL + zerolog + golang-migrate). Write SQL migrations for: teams(team_id uuid pk, team_name, created_by uuid, created_at), team_members(team_id, user_id, member_role, composite pk, CHECK role in manager|member), users_projection(user_id uuid pk, username, email, role, deleted_at nullable, updated_at). Add FKs from teams.created_by and team_members.user_id to users_projection.user_id. Start a Kafka consumer goroutine on topic 'user.events' (group 'core-user-projection', earliest offset) that upserts USER_CREATED/USER_UPDATED and soft-deletes on USER_DELETED. Do not use GORM AutoMigrate."

### Day 9 — JWT Middleware & Create Team

- [ ] JWT middleware: fetch **JWKS** from `JWKS_URL` on boot, cache keys by `kid`, verify RS256 signatures. Core holds **no** private key.
- [ ] Middleware also verifies `iss=auth-service`, `aud=seta`, `exp`, and checks Redis `jwt:blacklist:{jti}` (shared Redis with auth-service).
- [ ] `POST /teams` — manager only, creates team, adds creator as first manager. Validate `created_by` exists in `users_projection` (FK will also enforce).

**Prompt for Claude Code:**

> "Add JWT middleware to core-service using golang-jwt/jwt/v5. On boot, fetch JWKS from JWKS_URL (env), cache keys by kid, refresh on kid miss. Verify RS256, iss=auth-service, aud=seta, exp. Check Redis jwt:blacklist:{jti} (same Redis instance as auth-service). 401 on any failure. Implement POST /teams: role must be manager (403 else). created_by = sub from claims; must exist in users_projection (let FK enforce, but handle the error as 400). Insert creator into team_members as 'manager'."

### Day 10 — Add & Remove Members

- [ ] `POST /teams/:id/members` — manager of that team adds a member
- [ ] `DELETE /teams/:id/members/:userId` — manager removes a member

**Prompt for Claude Code:**

> "Implement POST /teams/:id/members and DELETE /teams/:id/members/:userId. Caller must be a manager of that team (check TeamMembers table). POST adds userId from body as 'member'. DELETE removes them. Return 403 if caller is not a manager of this team."

### Day 11 — Add & Remove Managers

- [ ] `POST /teams/:id/managers` — only original creator (Team.CreatedBy) can promote
- [ ] `DELETE /teams/:id/managers/:userId` — same restriction, cannot remove original creator

**Prompt for Claude Code:**

> "Add POST /teams/:id/managers and DELETE /teams/:id/managers/:userId. Only Team.CreatedBy == JWT user_id can call these. POST promotes a member to manager. DELETE demotes manager to member. Block removal of original creator."

### Day 12–14 — Testing

- [ ] Full flow: register users in auth-service → login → create team → add/remove members
- [ ] Verify member cannot create team (403)
- [ ] Verify manager cannot modify a team they don't belong to

**Learning checkpoint:** Two services running. JWT issued by auth-service is verified by core-service.

---

## Week 3 — Core Service: Assets & Sharing (Days 15–21)

**Goal:** Users own folders/notes. Sharing with RBAC works. Manager oversight works.

### Day 15 — Asset Model & CRUD

- [ ] `Asset`: AssetID, OwnerID, ParentID (nullable, self-ref), Type (folder|note), Title, Content, CreatedAt
- [ ] CRUD: `POST /assets`, `GET /assets/:id`, `PUT /assets/:id`, `DELETE /assets/:id`

**Prompt for Claude Code:**

> "Add Asset model to core-service (AssetID uuid pk, OwnerID uuid, ParentID uuid nullable self-referencing FK, Type string 'folder'|'note', Title string, Content text nullable, CreatedAt). CRUD endpoints: POST /assets, GET /assets/:id, PUT /assets/:id, DELETE /assets/:id. Only owner can update/delete."

### Day 16 — Access Control List

- [ ] `AssetACL`: AssetID, UserID, AccessLevel (read|write)
- [ ] Enforce ACL on every asset endpoint

**Prompt for Claude Code:**

> "Add AssetACL model (AssetID uuid, UserID uuid, AccessLevel string 'read'|'write', composite pk). Update GET /assets/:id to allow: owner OR any user with ACL entry. Update PUT/DELETE to allow: owner OR write ACL. Return 403 otherwise."

### Day 17 — Sharing Endpoint

- [ ] `POST /assets/:id/share` — `{userId, access: 'read'|'write'}`
- [ ] Folder sharing recursively shares all child notes
- [ ] `DELETE /assets/:id/share/:userId` — revoke access

**Prompt for Claude Code:**

> "Implement POST /assets/:id/share with body {userId, access: 'read'|'write'}. Only owner or write-access users can share. Upsert into AssetACL. If asset is a folder, recursively find all descendant notes and insert the same ACL for each. Add DELETE /assets/:id/share/:userId to revoke."

### Day 18 — Manager Oversight

- [ ] Managers have implicit read-only access to their team members' assets
- [ ] Check: is requester a manager of the asset owner's team? (query TeamMember in same DB)

**Prompt for Claude Code:**

> "Add manager oversight to asset access control in core-service. When a user would normally be denied, query TeamMembers to check if the requester is a manager in a team where the asset owner is a member. If yes, grant read-only access. Since team and asset data are in the same service, this is a direct DB query — no HTTP call needed."

### Day 19–20 — Bulk Import Integration & Testing

- [ ] End-to-end test: import 20 users via CSV → assign to teams → create assets → share
- [ ] Verify folder sharing cascades to child notes
- [ ] Verify manager can read member's assets without explicit share

### Day 21 — Polish

- [ ] Consistent error responses `{error: string, code: int}`
- [ ] Input validation on all endpoints

**Learning checkpoint:** Full asset CRUD, sharing, inheritance, and manager oversight working.

---

## Week 4 — Events, Caching & Docker (Days 22–30)

**Goal:** Kafka events, Redis caching, full containerization with logging.

### Day 22 — Kafka Setup

- [ ] Add Kafka + Zookeeper to docker-compose
- [ ] Topics: `team.activity`, `asset.changes`
- [ ] Shared Kafka producer helper in each service

**Prompt for Claude Code:**

> "Add Kafka and Zookeeper to docker-compose.yml (confluentinc/cp-kafka, confluentinc/cp-zookeeper, port 9092). Write a KafkaProducer struct using segmentio/kafka-go that publishes JSON messages to a named topic. Add it to both auth-service and core-service."

### Day 23 — Team Events

- [ ] core-service publishes to `team.activity` after each team mutation
- [ ] Events: `TEAM_CREATED`, `MEMBER_ADDED`, `MEMBER_REMOVED`, `MANAGER_ADDED`, `MANAGER_REMOVED`

**Prompt for Claude Code:**

> "In core-service, publish a Kafka message to 'team.activity' after each team mutation. Message format: {event: 'MEMBER_ADDED', team_id: '...', user_id: '...', timestamp: '...'}. Inject KafkaProducer into the team service layer and call it after each successful DB write."

### Day 24 — Asset Events & Audit Consumer

- [ ] core-service publishes to `asset.changes`: `NOTE_CREATED`, `NOTE_UPDATED`, `FOLDER_DELETED`, `FOLDER_SHARED`
- [ ] Start a consumer goroutine on startup — logs events to `audit_log` table

**Prompt for Claude Code:**

> "Add Kafka publishing to core-service for topic 'asset.changes' (NOTE_CREATED, NOTE_UPDATED, FOLDER_DELETED, FOLDER_SHARED). Also start a consumer goroutine on startup (consumer group 'audit') that reads from both 'team.activity' and 'asset.changes' and inserts each event into an audit_log table (event_id uuid, event_type string, payload jsonb, received_at timestamp)."

### Day 25 — Auth Events

- [ ] auth-service publishes `USER_CREATED`, `USER_UPDATED`, `USER_DELETED`, and `USERS_IMPORTED` events. The first three must carry **full projection fields** (`user_id, username, email, role`) — core-service uses them as the source of truth for `users_projection`.
- [ ] For `USERS_IMPORTED`, also emit one `USER_CREATED` per successfully-imported row so the projection stays in sync.
- [ ] Note: this is the payoff moment for the `users_projection` introduced on Day 8 — verify that a user created in auth-service shows up in `core-db.users_projection` within ~1s, and that a team referencing that user can now be created.

**Prompt for Claude Code:**

> "In auth-service, publish to Kafka topic 'user.events': USER_CREATED after POST /register ({user_id, username, email, role, timestamp}), USER_UPDATED / USER_DELETED for future admin endpoints (stub for now), and USERS_IMPORTED after POST /import-users with succeeded/failed counts. During import, also emit one USER_CREATED per successful row so the core-service users_projection stays in sync. Use the same KafkaProducer struct."

### Day 26 — Redis Caching

- [ ] Add Redis to docker-compose
- [ ] Cache team member list: `team:{teamId}:members` (invalidate on MEMBER_ADDED/REMOVED)
- [ ] Cache asset metadata: `asset:{assetId}` (invalidate on asset.changes)
- [ ] Cache ACL: `asset:{assetId}:acl` (invalidate on share/revoke)

**Prompt for Claude Code:**

> "Add Redis to docker-compose. In core-service, cache GET /teams/:id/members at 'team:{teamId}:members' (5-min TTL, go-redis/v9). Cache GET /assets/:id at 'asset:{assetId}'. Cache ACL at 'asset:{assetId}:acl'. In the audit consumer, invalidate relevant Redis keys when MEMBER_ADDED, MEMBER_REMOVED, NOTE_UPDATED, or FOLDER_SHARED events arrive."

### Day 27 — Dockerfiles

- [ ] Multi-stage Dockerfile for auth-service and core-service
- [ ] Update docker-compose to build and run both services

**Prompt for Claude Code:**

> "Write multi-stage Dockerfiles for auth-service (port 8081) and core-service (port 8082). Stage 1: golang:1.22-alpine, go build -o /app ./cmd. Stage 2: alpine:latest, copy binary. Update docker-compose.yml to build both services with env vars: DB_URL, KAFKA_BROKERS, REDIS_URL, JWT_SECRET, PORT."

### Day 28 — Loki + Promtail

- [ ] Add Loki and Promtail to docker-compose
- [ ] Configure Promtail to scrape Docker stdout logs
- [ ] Verify zerolog JSON appears in Loki

**Prompt for Claude Code:**

> "Add Grafana Loki and Promtail to docker-compose.yml. Write promtail-config.yml that scrapes Docker container stdout. Both services use zerolog for structured JSON output. Show the full promtail config and docker-compose additions."

### Day 29 — Integration Testing

- [ ] `docker-compose up` — everything starts cleanly
- [ ] Full end-to-end: register → login → create team → create asset → share → check Kafka events → check Redis cache hits → check Loki logs

### Day 30 — Tech Documentation

- [ ] Write 500–800 word tech decision doc

**Prompt for Claude Code:**

> "Write a 600-word technical documentation section for the SETA project. Explain: why 2 services over 3 (team+asset coupling), why REST over gRPC, why PostgreSQL over MongoDB, why Kafka over RabbitMQ (event replay, durability), why Redis for caching. Write it as a junior developer presenting to a tech lead."

**Learning checkpoint:** Full system running with one `docker-compose up`.

---

## Time Budget Summary

| Week   | Focus                                           | Days       |
| ------ | ----------------------------------------------- | ---------- |
| Week 1 | Auth Service (register, login, JWT, CSV import) | Days 1–7   |
| Week 2 | Core Service: Team management                   | Days 8–14  |
| Week 3 | Core Service: Assets, sharing, RBAC             | Days 15–21 |
| Week 4 | Kafka, Redis, Docker, Loki                      | Days 22–30 |

---

## General Prompting Tips for Claude Code

1. **One task at a time** — one endpoint or one layer per prompt works best.
2. **State what's already done:** "auth-service is done. core-service has teams working. Now add asset CRUD."
3. **Specify your stack every session:** "Using Go, Gin, GORM + PostgreSQL, segmentio/kafka-go, go-redis/v9, zerolog"
4. **Ask for tests:** "...and include unit tests for the service layer with mocked repository"
5. **Ask for explanation when learning:** "Implement X and explain why you structured it this way"
6. **When stuck:** "Here is my code [paste] and here is the error [paste]. What's wrong?"

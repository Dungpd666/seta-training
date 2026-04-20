# SETA System Design

## Table of Contents
1. [Requirements](#1-requirements)
2. [High-Level Architecture](#2-high-level-architecture)
3. [Database Design](#3-database-design)
4. [API Design](#4-api-design)
5. [Auth Flow](#5-auth-flow)
6. [Event Flow](#6-event-flow)
7. [Caching Strategy](#7-caching-strategy)
8. [Deployment](#8-deployment)

---

## 1. Requirements

### Functional Requirements

**Auth & User Management**
- Users can register with username, email, password, and role (manager or member)
- Users can log in and receive a JWT token
- Users can log out
- Admins can list all users
- Managers can bulk import users via CSV file (processed concurrently)

**Team Management**
- Managers can create teams
- Managers can add and remove members from their own teams
- Only the original team creator can promote or demote other managers
- Members cannot create or manage teams
- Managers can only manage teams they belong to

**Asset Management**
- Users can create folders and notes
- Users can read, update, and delete their own assets
- Users can share folders or notes with other users (read or write access)
- Sharing a folder automatically shares all notes inside it
- Users can revoke sharing access
- Managers have implicit read-only access to their team members' assets

### Non-Functional Requirements

| Property | Target |
|---|---|
| **Scale** | Thousands of concurrent users |
| **Availability** | 24/7 uptime |
| **Latency** | All API responses under 200ms (p99) |
| **Consistency** | Eventual consistency — cache may lag writes by a short window (seconds) |
| **Security** | JWT-based auth on all non-public endpoints; bcrypt password hashing |

---

## 2. High-Level Architecture

```
                        Clients (Web / Mobile / API)
                                    │
                          ┌─────────▼─────────┐
                          │    HTTP Requests   │
                          └────────┬──────────┘
                 ┌─────────────────┼──────────────────┐
                 │                                     │
        ┌────────▼────────┐                  ┌────────▼────────┐
        │  auth-service   │                  │  core-service   │
        │    :8081        │                  │    :8082        │
        │                 │                  │                 │
        │ /register       │                  │ /teams          │
        │ /login          │── JWKS (pubkey) ▶│ /assets         │
        │ /refresh        │    RS256         │ /assets/:id/    │
        │ /logout         │                  │    share        │
        │ /users          │                  │                 │
        │ /import-users   │                  │                 │
        └────────┬────────┘                  └────────┬────────┘
                 │                                     │
                 │            ┌────────────────────────┤
                 │            │                        │
        ┌────────▼────────┐   │               ┌────────▼────────┐
        │   auth-db       │   │               │   core-db       │
        │  (PostgreSQL)   │   │               │  (PostgreSQL)   │
        │                 │   │               │                 │
        │ users           │   │               │ teams           │
        └─────────────────┘   │               │ team_members    │
                              │               │ assets          │
                              │               │ asset_acl       │
                              │               │ audit_log       │
                              │               └─────────────────┘
                              │
                 ┌────────────▼────────────┐
                 │         Kafka           │
                 │                         │
                 │  topics:                │
                 │  - user.events          │
                 │  - team.activity        │
                 │  - asset.changes        │
                 └────────────┬────────────┘
                              │
                 ┌────────────▼────────────┐
                 │          Redis          │
                 │                         │
                 │  team:{id}:members      │
                 │  asset:{id}             │
                 │  asset:{id}:acl         │
                 └─────────────────────────┘
```

### Services

| Service | Responsibility | Port |
|---|---|---|
| `auth-service` | Identity — registration, login, JWT issuance, bulk import | 8081 |
| `core-service` | Collaboration — team management, asset CRUD, sharing, RBAC | 8082 |

**Why 2 services:**
- `auth-service` owns identity. Everything else just trusts the JWT — no need to know how auth works internally.
- `core-service` merges teams and assets because they are tightly coupled: manager oversight requires team membership lookups on every asset access check. Keeping them in the same service makes this a direct DB query instead of a cross-service HTTP call.

---

## 3. Database Design

### auth-db

```
┌──────────────────────────────────────────┐
│                  users                   │
├──────────────┬───────────────────────────┤
│ user_id      │ UUID PRIMARY KEY          │
│ username     │ VARCHAR(100) NOT NULL     │
│ email        │ VARCHAR(255) UNIQUE       │
│ password_hash│ VARCHAR(255) NOT NULL     │
│ role         │ VARCHAR(20) NOT NULL      │
│              │ CHECK (role IN            │
│              │  ('manager','member'))    │
│ created_at   │ TIMESTAMP DEFAULT NOW()  │
└──────────────┴───────────────────────────┘

┌──────────────────────────────────────────┐
│             refresh_tokens               │
├──────────────┬───────────────────────────┤
│ jti          │ UUID PRIMARY KEY          │
│ user_id      │ UUID NOT NULL             │
│ expires_at   │ TIMESTAMP NOT NULL        │
│ revoked      │ BOOLEAN DEFAULT FALSE     │
│ created_at   │ TIMESTAMP DEFAULT NOW()  │
└──────────────┴───────────────────────────┘
-- Refresh tokens are persisted so logout + rotation are real.
-- Access-token revocation (short-lived, 15m) uses a Redis jti
-- blacklist with TTL = remaining exp.

┌──────────────────────────────────────────┐
│           schema_migrations              │   (managed by
├──────────────┬───────────────────────────┤    golang-migrate;
│ version      │ BIGINT PRIMARY KEY        │    NOT GORM
│ dirty        │ BOOLEAN NOT NULL          │    AutoMigrate)
└──────────────┴───────────────────────────┘
```

### core-db

```
┌──────────────────────────────────────────┐
│                  teams                   │
├──────────────┬───────────────────────────┤
│ team_id      │ UUID PRIMARY KEY          │
│ team_name    │ VARCHAR(100) NOT NULL     │
│ created_by   │ UUID NOT NULL             │
│ created_at   │ TIMESTAMP DEFAULT NOW()  │
└──────────────┴───────────────────────────┘

┌──────────────────────────────────────────┐
│              team_members                │
├──────────────┬───────────────────────────┤
│ team_id      │ UUID REFERENCES teams     │
│ user_id      │ UUID NOT NULL             │
│ member_role  │ VARCHAR(20) NOT NULL      │
│              │ CHECK (member_role IN     │
│              │  ('manager','member'))    │
│ PRIMARY KEY  │ (team_id, user_id)        │
└──────────────┴───────────────────────────┘

┌──────────────────────────────────────────┐
│                  assets                  │
├──────────────┬───────────────────────────┤
│ asset_id     │ UUID PRIMARY KEY          │
│ owner_id     │ UUID NOT NULL             │
│ parent_id    │ UUID REFERENCES assets    │
│              │ (nullable — NULL = root)  │
│ type         │ VARCHAR(10) NOT NULL      │
│              │ CHECK (type IN            │
│              │  ('folder','note'))       │
│ title        │ VARCHAR(255) NOT NULL     │
│ content      │ TEXT (nullable)           │
│ created_at   │ TIMESTAMP DEFAULT NOW()  │
│ updated_at   │ TIMESTAMP DEFAULT NOW()  │
└──────────────┴───────────────────────────┘

┌──────────────────────────────────────────┐
│               asset_acl                  │
├──────────────┬───────────────────────────┤
│ asset_id     │ UUID REFERENCES assets    │
│ user_id      │ UUID NOT NULL             │
│ access_level │ VARCHAR(10) NOT NULL      │
│              │ CHECK (access_level IN    │
│              │  ('read','write'))        │
│ PRIMARY KEY  │ (asset_id, user_id)       │
└──────────────┴───────────────────────────┘

┌──────────────────────────────────────────┐
│               audit_log                  │
├──────────────┬───────────────────────────┤
│ event_id     │ UUID PRIMARY KEY          │
│ event_type   │ VARCHAR(50) NOT NULL      │
│ payload      │ JSONB NOT NULL            │
│ received_at  │ TIMESTAMP DEFAULT NOW()  │
└──────────────┴───────────────────────────┘

┌──────────────────────────────────────────┐
│             users_projection             │   Local read-model
├──────────────┬───────────────────────────┤   built from the
│ user_id      │ UUID PRIMARY KEY          │   user.events Kafka
│ username     │ VARCHAR(100) NOT NULL     │   topic. Enables FK
│ email        │ VARCHAR(255) NOT NULL     │   constraints on
│ role         │ VARCHAR(20) NOT NULL      │   teams.created_by,
│ deleted_at   │ TIMESTAMP NULL            │   assets.owner_id,
│ updated_at   │ TIMESTAMP DEFAULT NOW()  │   asset_acl.user_id
└──────────────┴───────────────────────────┘   — no cross-service
                                                HTTP required.
```

### Entity Relationships

```
users (auth-db)
  │
  │ user_id referenced across services via JWT claims
  │
  ├──▶ teams.created_by (core-db)
  ├──▶ team_members.user_id
  ├──▶ assets.owner_id
  └──▶ asset_acl.user_id

teams ──< team_members
assets ──< asset_acl
assets ──< assets (self-ref: parent_id for folder/note hierarchy)
```

---

## 4. API Design

### auth-service (port 8081)

| Method | Endpoint | Auth | Description |
|---|---|---|---|
| POST | `/register` | None | Register a new user |
| POST | `/login` | None | Login, returns access + refresh tokens |
| POST | `/refresh` | Refresh token | Rotate: new access token, rotated refresh token |
| POST | `/logout` | JWT | Revoke refresh token + blacklist access jti in Redis |
| GET | `/users` | JWT | List all users |
| POST | `/import-users` | JWT (manager) | Bulk import users from CSV |
| GET | `/.well-known/jwks.json` | None | RS256 public key set (consumed by core-service) |
| GET | `/health` | None | Health check |

**POST /register**
```json
Request:  { "username": "alice", "email": "alice@x.com", "password": "secret", "role": "manager" }
Response: 201 { "user_id": "uuid", "username": "alice", "email": "alice@x.com", "role": "manager" }
Errors:   409 if email already exists
```

**POST /login**
```json
Request:  { "email": "alice@x.com", "password": "secret" }
Response: 200 {
            "access_token":  "eyJ...",   // RS256, 15-min exp
            "refresh_token": "eyJ...",   // 7-day exp, rotated on use
            "token_type":    "Bearer"
          }
Errors:   401 if credentials invalid
```

**POST /refresh**
```json
Request:  { "refresh_token": "eyJ..." }
Response: 200 { "access_token": "eyJ...", "refresh_token": "eyJ..." }
Errors:   401 if refresh token expired, revoked, or already rotated
```

**POST /logout**
```
Headers: Authorization: Bearer <access_token>
Body:    { "refresh_token": "eyJ..." }
Effect:  - Marks refresh_tokens.revoked = TRUE
         - Blacklists the access token's jti in Redis with TTL = remaining exp
Response: 204
```

**POST /import-users**
```
Request:  multipart/form-data, field "file" = CSV
          CSV columns: username, email, password, role
Response: 200 { "succeeded": 18, "failed": 2, "errors": [{"row": 3, "reason": "email already exists"}] }
```

---

### core-service (port 8082)

**Teams**

| Method | Endpoint | Auth | Description |
|---|---|---|---|
| POST | `/teams` | JWT (manager) | Create a team |
| GET | `/teams/:id` | JWT | Get team details |
| GET | `/teams/:id/members` | JWT | List team members |
| POST | `/teams/:id/members` | JWT (team manager) | Add a member |
| DELETE | `/teams/:id/members/:userId` | JWT (team manager) | Remove a member |
| POST | `/teams/:id/managers` | JWT (creator) | Promote to manager |
| DELETE | `/teams/:id/managers/:userId` | JWT (creator) | Demote manager |

**POST /teams**
```json
Request:  { "team_name": "Backend Team" }
Response: 201 { "team_id": "uuid", "team_name": "Backend Team", "created_by": "uuid" }
Errors:   403 if caller is not a manager
```

**Assets**

| Method | Endpoint | Auth | Description |
|---|---|---|---|
| POST | `/assets` | JWT | Create folder or note |
| GET | `/assets/:id` | JWT | Get asset (owner, ACL, or team manager) |
| PUT | `/assets/:id` | JWT | Update asset (owner or write ACL) |
| DELETE | `/assets/:id` | JWT | Delete asset (owner only) |
| POST | `/assets/:id/share` | JWT | Share asset with a user |
| DELETE | `/assets/:id/share/:userId` | JWT | Revoke access |

**POST /assets**
```json
Request:  { "type": "folder", "title": "My Folder", "parent_id": null }
          { "type": "note", "title": "My Note", "content": "Hello", "parent_id": "folder-uuid" }
Response: 201 { "asset_id": "uuid", "type": "folder", "title": "My Folder", "owner_id": "uuid" }
```

**POST /assets/:id/share**
```json
Request:  { "user_id": "uuid", "access": "read" }
Response: 200 { "message": "shared successfully" }
```

---

## 5. Auth Flow

```
Client          auth-service                core-service
  │                  │                            │
  │── POST /login ──▶│                            │
  │                  │ verify bcrypt              │
  │                  │ sign access (RS256, 15m)   │
  │                  │ sign+store refresh (7d)    │
  │◀── {access,      │                            │
  │     refresh} ───│                            │
  │                  │                            │
  │           (core boots, fetches JWKS once)     │
  │                  │◀── GET /.well-known/jwks ─│
  │                  │─── {keys: [pubkey]} ─────▶│
  │                  │                            │
  │── GET /assets/:id (Bearer access) ───────────▶│
  │                  │                            │ verify RS256 with pubkey
  │                  │                            │ check jti not in Redis blacklist
  │                  │                            │ extract user_id, role
  │                  │                            │ check ownership / ACL
  │◀────────────────────────── 200 { asset } ────│
  │                  │                            │
  │── POST /refresh ▶│  rotate refresh,           │
  │                  │  issue new access          │
  │◀── {access,      │                            │
  │     refresh} ───│                            │
  │                  │                            │
  │── POST /logout ─▶│  revoke refresh,           │
  │                  │  blacklist access jti      │
  │◀──── 204 ───────│                            │
```

**Access-token Payload (RS256)**
```json
{
  "sub":     "user-uuid",
  "role":    "manager",
  "jti":     "token-uuid",     // used for Redis blacklist on logout
  "iss":     "auth-service",
  "aud":     "seta",
  "iat":     1234567890,
  "exp":     1234568790         // 15 min
}
```

**Refresh-token Payload (RS256)**
```json
{ "sub": "user-uuid", "jti": "refresh-uuid", "typ": "refresh", "exp": 1235172690 }
```

- **Signing:** RS256. `auth-service` holds the **private key**; `core-service` fetches the **public key** from `/.well-known/jwks.json` on boot (cached, refreshed on `kid` miss).
  Rationale: with a shared HS256 secret, `core-service` could forge tokens. Asymmetric signing makes `auth-service` the only issuer.
- **Access token expiry:** 15 minutes. Short-lived so a stolen token has a tight blast radius.
- **Refresh token expiry:** 7 days, persisted in `refresh_tokens` and rotated on every `/refresh`. Reuse of an already-rotated refresh token revokes the whole chain (detected theft).
- **Logout:** revokes the refresh token (DB) **and** blacklists the current access-token `jti` in Redis with TTL = remaining exp. `core-service` middleware checks Redis on every request. No infinite-growth blacklist — entries expire on their own.
- **Key rotation:** `auth-service` can publish multiple keys in JWKS keyed by `kid`; `core-service` picks the right one per token.

---

## 6. Event Flow

### Topics

| Topic | Publisher | Consumer | Purpose |
|---|---|---|---|
| `user.events` | auth-service | core-service (projection + audit) | Maintain local `users_projection` read-model and audit log |
| `team.activity` | core-service | core-service (audit) | Track team mutations |
| `asset.changes` | core-service | core-service (cache invalidation + audit) | Track asset mutations |

### Event Schemas

**user.events**
```json
{ "event": "USER_CREATED", "user_id": "uuid", "username": "alice", "email": "alice@x.com", "role": "manager", "timestamp": "..." }
{ "event": "USER_UPDATED", "user_id": "uuid", "username": "alice", "email": "alice@x.com", "role": "manager", "timestamp": "..." }
{ "event": "USER_DELETED", "user_id": "uuid", "timestamp": "..." }
{ "event": "USERS_IMPORTED", "succeeded": 18, "failed": 2, "timestamp": "..." }
```
`USER_CREATED` / `USER_UPDATED` / `USER_DELETED` are upserts/tombstones into `users_projection` in core-db. This lets `teams.created_by`, `assets.owner_id`, and `asset_acl.user_id` be validated **locally** — no sync HTTP call to auth-service on every write.

**team.activity**
```json
{ "event": "TEAM_CREATED",    "team_id": "uuid", "user_id": "uuid", "timestamp": "..." }
{ "event": "MEMBER_ADDED",    "team_id": "uuid", "user_id": "uuid", "timestamp": "..." }
{ "event": "MEMBER_REMOVED",  "team_id": "uuid", "user_id": "uuid", "timestamp": "..." }
{ "event": "MANAGER_ADDED",   "team_id": "uuid", "user_id": "uuid", "timestamp": "..." }
{ "event": "MANAGER_REMOVED", "team_id": "uuid", "user_id": "uuid", "timestamp": "..." }
```

**asset.changes**
```json
{ "event": "NOTE_CREATED",   "asset_id": "uuid", "owner_id": "uuid", "timestamp": "..." }
{ "event": "NOTE_UPDATED",   "asset_id": "uuid", "timestamp": "..." }
{ "event": "FOLDER_DELETED", "asset_id": "uuid", "timestamp": "..." }
{ "event": "FOLDER_SHARED",  "asset_id": "uuid", "shared_with": "uuid", "access": "read", "timestamp": "..." }
```

### Event Flow Diagram

```
auth-service                     core-service
     │                                │
     │── USER_CREATED ──▶ user.events │
     │── USERS_IMPORTED ─▶ user.events│
     │                                │
     │                                │── TEAM_CREATED ──▶ team.activity
     │                                │── MEMBER_ADDED ──▶ team.activity
     │                                │── NOTE_CREATED ──▶ asset.changes
     │                                │── FOLDER_SHARED ─▶ asset.changes
     │                                │
     │                    ┌───────────┘
     │                    │ consumer goroutine (core-service)
     │                    │
     │                    ├── team.activity  ──▶ invalidate Redis team cache
     │                    │                 ──▶ insert audit_log
     │                    │
     │                    └── asset.changes ──▶ invalidate Redis asset cache
     │                                     ──▶ insert audit_log
```

---

## 7. Caching Strategy

### What is cached

| Cache Key | Value | TTL | Invalidated By |
|---|---|---|---|
| `team:{teamId}:members` | List of team members with roles | 5 min | `MEMBER_ADDED`, `MEMBER_REMOVED`, `MANAGER_ADDED`, `MANAGER_REMOVED` |
| `asset:{assetId}` | Asset metadata (title, type, owner) | 5 min | `NOTE_UPDATED`, `FOLDER_DELETED` |
| `asset:{assetId}:acl` | List of users with access levels | 5 min | `FOLDER_SHARED`, and on share revocation |

### Cache Flow (Read)

```
Request GET /assets/:id
        │
        ▼
Check Redis key "asset:{id}"
        │
   ┌────┴────┐
 HIT        MISS
   │          │
   │          ▼
   │     Query PostgreSQL
   │          │
   │          ▼
   │     Write to Redis (TTL 5 min)
   │          │
   └────┬─────┘
        ▼
   Return response
```

### Cache Invalidation Flow

```
PUT /assets/:id (update note)
        │
        ▼
Update PostgreSQL
        │
        ▼
Publish NOTE_UPDATED to Kafka
        │
        ▼
Consumer goroutine receives event
        │
        ▼
DELETE Redis key "asset:{id}"
DELETE Redis key "asset:{id}:acl"
```

### Why eventual consistency is acceptable here
- A stale cache window of a few seconds is acceptable for asset reads
- Team member list cache lag is tolerable — access control falls back to DB on cache miss for sensitive operations
- Audit log is written from Kafka events, ensuring no events are lost even if cache invalidation is delayed

---

## 8. Deployment

### docker-compose services

```
┌─────────────────────────────────────────────────────────────┐
│                      docker-compose                         │
│                                                             │
│  ┌──────────────┐   ┌──────────────┐   ┌────────────────┐  │
│  │ auth-service │   │ core-service │   │   zookeeper    │  │
│  │ :8081        │   │ :8082        │   │   :2181        │  │
│  └──────┬───────┘   └──────┬───────┘   └───────┬────────┘  │
│         │                  │                   │            │
│  ┌──────▼───────┐   ┌──────▼───────┐   ┌───────▼────────┐  │
│  │   auth-db    │   │   core-db    │   │     kafka      │  │
│  │ (PostgreSQL) │   │ (PostgreSQL) │   │     :9092      │  │
│  │ :5432        │   │ :5433        │   └────────────────┘  │
│  └──────────────┘   └──────────────┘                        │
│                                                             │
│  ┌──────────────┐   ┌──────────────┐   ┌────────────────┐  │
│  │    redis     │   │     loki     │   │   promtail     │  │
│  │    :6379     │   │    :3100     │   │                │  │
│  └──────────────┘   └──────────────┘   └────────────────┘  │
└─────────────────────────────────────────────────────────────┘
```

### Environment Variables

**auth-service**
```
PORT=8081
DB_URL=postgres://user:pass@auth-db:5432/authdb
JWT_PRIVATE_KEY_PATH=/secrets/jwt_rs256.pem       # RS256 private key (signing)
JWT_PUBLIC_KEY_PATH=/secrets/jwt_rs256.pub        # served via /.well-known/jwks.json
JWT_KEY_ID=2026-04                                 # kid header, rotate by changing
ACCESS_TOKEN_TTL=15m
REFRESH_TOKEN_TTL=168h
REDIS_URL=redis:6379                               # for access-jti blacklist on logout
KAFKA_BROKERS=kafka:9092
```

**core-service**
```
PORT=8082
DB_URL=postgres://user:pass@core-db:5433/coredb
JWKS_URL=http://auth-service:8081/.well-known/jwks.json   # fetched on boot, cached
JWT_ISSUER=auth-service
JWT_AUDIENCE=seta
KAFKA_BROKERS=kafka:9092
REDIS_URL=redis:6379
```
Note: `core-service` holds **no signing key** — only the public key it fetches from `auth-service`. It cannot mint tokens.

### Service Dependencies (startup order)

```
zookeeper
    └──▶ kafka
auth-db ──▶ auth-service
core-db ──▶ core-service
redis   ──▶ core-service
kafka   ──▶ auth-service
kafka   ──▶ core-service
```

### Dockerfile pattern (each service)

```dockerfile
# Stage 1: Build
FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o /service ./cmd

# Stage 2: Run
FROM alpine:latest
COPY --from=builder /service /service
EXPOSE 8081
CMD ["/service"]
```

### Logging

- All services use `zerolog` for structured JSON output to stdout
- Promtail scrapes container stdout and ships to Loki
- Loki stores and indexes logs for querying
- Each log line includes: `level`, `service`, `timestamp`, `trace_id`, `message`

---

## Design Decisions Summary

| Decision | Choice | Reason |
|---|---|---|
| 2 services over 3 | auth + core | Teams and assets are tightly coupled via manager oversight — same service avoids cross-service HTTP calls on every permission check |
| REST over gRPC | REST | Simpler, easier to test with curl/Postman, sufficient for this scale |
| PostgreSQL over MongoDB | PostgreSQL | Relational data (ACL, team membership) benefits from SQL joins and foreign key constraints |
| Kafka over RabbitMQ | Kafka | Durable event log, consumer group replay, industry standard |
| Eventual consistency | Redis + Kafka invalidation | Acceptable for this domain; simplifies architecture vs. distributed transactions |
| JWT signing | **RS256 (asymmetric) + JWKS** | Only `auth-service` can mint tokens; `core-service` verifies with public key. Prevents a compromised core-service from forging identities |
| Token lifetime | Access 15m + rotating refresh 7d | Short access TTL bounds stolen-token damage; refresh rotation enables real server-side logout |
| Logout | Redis jti blacklist + refresh revoke | Stateless JWT has no native revocation — blacklist entries TTL-out on their own, so no unbounded growth |
| Schema migrations | **golang-migrate**, not GORM AutoMigrate | AutoMigrate silently drifts, can't roll back, won't create required indexes; versioned SQL migrations are auditable |
| Cross-service user refs | **`users_projection` in core-db, built from `user.events`** | FK-enforceable locally; no sync HTTP call to auth-service on every team/asset write; handles user deletion gracefully |

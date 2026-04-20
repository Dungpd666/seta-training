# SETA — Design Review & Recommendations

Review of `docs/PLAN.md` and `docs/SYSTEM_DESIGN.md`. The first section is the full list of issues raised; the second section is the deep-dive explanation for issues 1–4 (which have been applied to `PLAN.md` and `SYSTEM_DESIGN.md`).

---

## Part 1 — All Issues Raised

### 1. JWT: switch HS256 → RS256 (asymmetric)
The shared-secret model means `core-service` can *forge* tokens, not just verify them. Let `auth-service` sign with a private key and publish a JWKS endpoint — `core-service` only ever holds the public key. Same code complexity, much better blast radius.

### 2. Real logout + short access tokens
A 24h stateless token with client-side-only logout is a compliance/security problem (stolen token = 24h access). Options:
- 15-min access token + refresh token stored in Redis (revocable)
- Or a Redis JTI blacklist checked in middleware

Don't ship logout as a no-op.

### 3. Replace GORM AutoMigrate with versioned migrations
`golang-migrate` or `goose`. AutoMigrate is fine in dev but it silently drifts in prod, can't roll back, and won't add the indexes you need.

### 4. The "user_id exists in core-db" problem is unsolved
`core-service` stores `teams.created_by`, `assets.owner_id`, `asset_acl.user_id` — all UUIDs pointing at a DB it can't query. Today nothing validates them. Three options:
- **Projection via `user.events`**: consume `USER_CREATED` into a local `users_projection` table in core-db. Then FKs work, and you get a graceful answer for "what about deleted users."
- Sync call to auth at write time (couples services)
- Ignore and accept dangling refs (current plan)

Recommended: the projection — it's the reason you already have `user.events`.

### 5. Manager oversight needs indexes, not just a query
On every asset GET you'll query `team_members` to check if the requester is a manager of a team the owner belongs to. Add:
- `idx(team_members.user_id, member_role)` — find teams where I'm manager
- `idx(team_members.team_id, user_id)` — find if user is in team

Consider materializing `manager_can_see(user_id, owned_by)` if it gets hot.

### 6. Folder-share cascade needs to be async
Sharing a folder with 10k descendant notes inside an HTTP request will time out and lock rows. Publish a `FOLDER_SHARE_REQUESTED` event, do it in a worker, return 202.

### 7. Transactional outbox for Kafka
`publish-after-db-commit` is a well-known footgun (DB commits, broker down → lost event; or broker ok but response dropped → duplicate event). Write the event to an `outbox` table in the same tx, publish from a relay. Especially matters because your cache invalidation depends on those events arriving.

### 8. Operational gaps worth filling in Week 4
- **Rate limiting on `/login`** (brute-force) — easy win with a Redis token bucket
- **Pagination** on `/users`, `/teams/:id/members`, asset listings — the plan has none
- **`/healthz` vs `/readyz`** split (ready = DB + Kafka + Redis reachable)
- **Request ID propagation** across both services for log correlation

### What not to change
- 2 services vs 3: the coupling argument (manager oversight = team+asset in same service) is correct.
- Postgres over Mongo, REST over gRPC, Kafka over RabbitMQ — all fine for this scale.
- Separate DBs per service — keep it.

---

## Part 2 — Deep Dive on Issues 1–4 (applied to PLAN.md + SYSTEM_DESIGN.md)

### Problem 1 — RS256 (asymmetric) instead of HS256 shared secret

**What was wrong.** The original design had both services holding the same `JWT_SECRET`. With a symmetric secret, any service that can *verify* a token can also *mint* one. That means if `core-service` is ever compromised (SQL injection, dependency CVE, leaked env var), the attacker can forge arbitrary identity tokens — including for admins — and walk into `auth-service` as anyone.

**The fix.** `auth-service` holds an RSA **private key** and is the only thing that can sign. `core-service` only gets the **public key**, fetched from `auth-service/.well-known/jwks.json` on boot. Public keys can't mint. Same `golang-jwt` library, same token shape, strictly smaller blast radius. Bonus: key rotation becomes possible — publish a new `kid` in JWKS and old tokens still validate against the old key until they expire.

---

### Problem 2 — Short access tokens + refresh tokens + real logout

**What was wrong.** A 24-hour stateless JWT with client-side-only logout means "logout" is a lie. If a token leaks (XSS, stolen laptop, logged to Sentry, proxied through a malicious extension), the attacker has a full day of access and there is **nothing** the server can do about it. That is not acceptable for a system with manager oversight and asset RBAC.

**The fix (two layers).**
- **Access token: 15 min, RS256.** A leaked token is useful for 15 min, not 24 h. That's the core of the damage reduction.
- **Refresh token: 7 d, RS256, persisted in `refresh_tokens` table, rotated on every use.** Clients call `POST /refresh` to get a new access token. Since the refresh `jti` lives in the DB, `/logout` can truly revoke it. **Refresh-reuse detection**: if a token that was already rotated shows up again, someone has it who shouldn't — revoke the whole chain for that user.
- **Access revocation on logout:** push the access `jti` into Redis with TTL = remaining exp. Middleware checks the blacklist on every request. Because entries self-expire, the blacklist never grows unboundedly — the worst-case size is (logouts in last 15 min), not all-time.

---

### Problem 3 — `golang-migrate` instead of GORM `AutoMigrate`

**What was wrong.** `AutoMigrate` is a convenience that silently diffs your struct against the DB on every boot. In practice:
- It **won't drop** columns or rename them — your schema drifts.
- It **can't version** — no audit trail of what changed when.
- It **can't roll back** — if a deploy is bad you're stuck hand-fixing prod.
- It **won't create** the composite/partial indexes you need (e.g., `team_members(user_id, member_role)` for the manager-oversight hot path).
- It silently uses struct tag conventions; a typo = a missing constraint.

**The fix.** Versioned SQL files in `migrations/` (e.g., `000001_create_users.up.sql` + `.down.sql`), run via `golang-migrate` on boot (fail-fast). Plain SQL means CHECK constraints, FKs, and indexes are explicit and reviewable. GORM is still fine for **queries** — just not for schema.

---

### Problem 4 — `users_projection` in core-db, fed by `user.events`

**What was wrong.** The original design stored `teams.created_by`, `assets.owner_id`, `asset_acl.user_id` as UUIDs — but `core-service` has no access to the `users` table (different DB). Consequences:
- **No FK constraint is possible**, so `POST /teams` could be called with `created_by = <random-uuid>` and nothing would catch it.
- **User deletion has no defined semantics.** If a user is deleted in auth-db, all their assets and team memberships point into the void.
- **Any enrichment** ("show me the username of this asset's owner") would require a cross-service HTTP call per row — N+1 across service boundaries.

**The fix.** `core-db.users_projection` is a local read-model, built by consuming `user.events` from Kafka:
- `USER_CREATED` / `USER_UPDATED` → upsert
- `USER_DELETED` → tombstone (`deleted_at = NOW()`)

Now `core-db` can enforce FKs from `teams.created_by`, `team_members.user_id`, and `asset_acl.user_id` to `users_projection.user_id`. No synchronous call to `auth-service` on the hot path. The consumer starts from **earliest offset** so a fresh core-db backfills itself from the log — the Kafka topic is effectively the source of truth. This is the textbook CQRS-style local-projection pattern, and you already have the topic, so the cost is one consumer goroutine.

---

### Remaining coupling to notice

`core-service` now reads Redis for the access-token blacklist (shared instance with `auth-service`). That's a minor coupling but it's just a key-value check, no schema contract, so it's cheap. If you wanted to eliminate it later, you'd publish a `TOKEN_REVOKED` Kafka event and have each service maintain its own local blacklist — but for this scale, shared Redis is the right call.

---

## Status

| # | Issue | Status |
|---|---|---|
| 1 | RS256 + JWKS | Applied to `PLAN.md` (Day 3–4, 9) and `SYSTEM_DESIGN.md` (§4, §5, §8) |
| 2 | Short access + refresh + real logout | Applied — same locations + `refresh_tokens` table |
| 3 | `golang-migrate` over AutoMigrate | Applied — Day 1, 2, 8 + tech stack table |
| 4 | `users_projection` via `user.events` | Applied — Day 8, 25 + `SYSTEM_DESIGN.md` §3, §6 |
| 5 | Indexes for manager oversight | **Open** |
| 6 | Async folder-share cascade | **Open** |
| 7 | Transactional outbox for Kafka | **Open** |
| 8 | Rate-limit / pagination / `/readyz` / request-id | **Open** |

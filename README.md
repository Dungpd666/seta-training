# seta-training

Two Go microservices for the 2026 Golang Intern Course capstone.

## Prerequisites

- Docker & Docker Compose
- Go 1.25+
- OpenSSL (for RSA key generation)

## Quick Start

**1. Start infrastructure**

```bash
docker-compose up -d
```

Starts: PostgreSQL ×2 (ports 5480, 5433), Redis (6379), Kafka + Zookeeper (9092).

**2. Generate RSA keys** (auth-service only, one-time)

```bash
cd auth-service
openssl genpkey -algorithm RSA -pkeyopt rsa_keygen_bits:2048 -out jwt_rs256.pem
openssl rsa -in jwt_rs256.pem -pubout -out jwt_rs256.pub
```

**3. Configure environment**

```bash
# auth-service/.env
DB_URL=postgres://postgres:postgres@localhost:5480/authdb

# core-service/.env
DB_URL=postgres://postgres:postgres@localhost:5433/coredb
```

**4. Run services** (each in a separate terminal)

```bash
cd auth-service && go run ./cmd/main.go   # :8081
cd core-service && go run ./cmd/main.go   # :8082
```

Migrations run automatically on startup — no manual step needed.

## Run Tests

```bash
# All tests
go test ./...

# Single test with verbose output
go test ./internal/team/... -run TestCreateTeam_CreatorAutoAddedAsManager -v
```

## Services

| Service | Port | Responsibility |
|---|---|---|
| auth-service | 8081 | Registration, login, JWT issuance, user listing, CSV import |
| core-service | 8082 | Teams, assets (folders & notes), sharing, RBAC |

## Documentation

| File | Purpose |
|---|---|
| `docs/GOAL.md` | Project requirements and grading criteria |
| `docs/tech-stack.md` | Tech stack choices and trade-offs |
| `docs/ARCHITECTURE.md` | System architecture overview |

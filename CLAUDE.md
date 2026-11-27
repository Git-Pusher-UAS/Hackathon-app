# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Stack

Go backend ("SAI API"), module `backend-hackathon`: Fiber v2, GORM on PostgreSQL 17, JWT + Argon2id auth, TOTP MFA, Swagger via swag. `docs/commands-exec.md` (Spanish) is the team's full command guide.

## Commands

```bash
docker compose up -d db          # Postgres on host port 5433
go run ./cmd/migrate             # schemas + AutoMigrate + views + functional indexes (idempotent)
go run ./cmd/seed                # roles, permissions, INITIAL_ADMIN_* account (idempotent; needs migrate first)
go run ./cmd                     # API on $PORT

go build ./... && go vet ./... && gofmt -l cmd internal tests
go test ./...                                    # all unit tests
go test ./internal/services/user -run TestName   # single test

# smoke test against a running API (build tag "smoke")
SMOKE_BASE_URL=http://localhost:3000 SMOKE_TOKEN=<jwt> go test -tags smoke ./tests/smoke/ -v

swag init -g cmd/main.go -d . --parseDependency --parseInternal   # regenerate docs/ after changing annotations
```

Full Docker flow: `docker compose up -d --build`, then `docker compose --profile tools run --rm migrate` and `... run --rm seed`. On a fresh volume the order is migrate → seed → backend. Login is rate limited: 5 failures in 15 min returns 429.

## Architecture

- `cmd/main.go` wires everything: config → GORM → `UnitOfWork` → services → handlers → Fiber routes. All routes are registered there.
- Layers: `internal/handlers/*` (HTTP, DTO binding) → `internal/services/*` (business rules) → `internal/repository/repositories` (interfaces) implemented by `internal/repository/gorm` (`uow.go` is the transaction boundary that services receive). Models live in `internal/models`, split across Postgres schemas (e.g. `identidad`, `auditoria`).
- Two API surfaces with different error formats:
  - `/api/*` (legacy: login, me, logout, MFA, CSRF token) returns `{"error": "msg"}`.
  - `/api/v1/*` (users, case-files) returns `{"error": {"code", "message", "requestId"}}` via `internal/httpx`. It is entirely behind `AuthMiddleware`, writes need `X-CSRF-Token`, and it uses signed cursors (`httpx/cursor.go`) and ETags.
- Auth: `middleware.AuthMiddleware` validates the JWT *and* a revocable DB session. Permissions are checked with `RequirePermiso` / `casefile.Require(perm)`. A login with MFA active answers 403 with `challenge_token`. A failed login answers a generic 401 (it never reveals whether the user exists), not 400.
- Encryption: `internal/security` does envelope encryption (the `KEK_MASTER` key wraps per-record keys). `CURP_HMAC_KEY` is used for duplicate checks. If either is missing, the API still starts but the case-file/encrypted endpoints answer 503. `MFA_ENCRYPTION_KEY` and `JWT_SECRET` are mandatory.
- Every `internal/{handlers,services,middleware,models}` folder has subfolders per domain module (many still only have `.gitkeep`). New modules follow the `user` / `casefile` pattern.

## Config

Env vars are loaded from `.env` via godotenv (git-ignored). `.env.example` lists every key. Keys are standard base64: 32 bytes, 64 for `JWT_SECRET`. When adding a key, keep `.env.example`, `internal/config/config.go`, `.github/workflows/ci.yml` and `.github/scripts/ci-secrets.sh` in sync.

## Tests and CI

- Unit tests sit next to the code (`*_test.go`). Handlers are tested with Fiber + fakes.
- `tests/smoke/checks.go` holds the smoke checks as plain functions over a `Target`. `checks_test.go` unit-tests them against fake `httptest` servers, so they must fail on a broken backend. `smoke_test.go` runs them against a live API. Routes are overridable with `SMOKE_*` env vars.
- `.github/workflows/ci.yml` runs:
  - lint: gofmt, vet, golangci-lint, with only new issues reported on PRs;
  - unit tests with `-race` and the `COVERAGE_MIN` gate;
  - smoke: Postgres service → migrate → seed → API → admin login → smoke tests;
  - k6 (`tests/load/k6_load_test.js`), on push to `develop`/`main` only.

  Secrets are generated randomly per run by `.github/scripts/ci-secrets.sh`.

## Git workflow (from `git/GIT_FLOW.md`, Spanish)

- Run `python git/setup_git.py` once per clone. It sets `pull.rebase=false`, `pull.ff=false` and `merge.ff=false`, so every merge is an explicit merge commit. Don't rebase or fast-forward. (On Windows the script prints a `UnicodeEncodeError` at the end, but the config is already applied.)
- Branches: `main` (production, merged only from `develop` by the project lead) ← `develop` (integration) ← `feat/<kebab-case-name>`. Branch from an up-to-date `develop`, merge back into `develop` only, never directly into `main`, never work on `main`/`develop`.
- Commit messages use conventional prefixes in Spanish (`feat: ...`, `fix: ...`, `chore: ...`).

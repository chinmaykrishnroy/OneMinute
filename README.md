# 60 Second Internet

A real-time conversation application being built in vertical milestones. Branding is temporary; internal names use **encounter**.

**Current slice: Milestone 0.** Next.js shell, Go API, PostgreSQL/pgvector, Redis, independent Pion TURN, migrations and development tooling. Authentication, matchmaking and calls are not part of this foundation yet.

## Run locally

Requires Docker Engine/Desktop with Linux containers and Compose. Node 24 and Go 1.26.5 are used for host checks; containers include build tools. No Google credentials are needed for this milestone.

PowerShell, from this directory:

```powershell
./scripts/dev.ps1 init
./scripts/dev.ps1 up
./scripts/dev.ps1 smoke
```

Linux/macOS (or any shell with Node):

```sh
node scripts/init-env.mjs
docker compose up --build --detach --wait --wait-timeout 180
node scripts/smoke.mjs
```

Open [localhost:3000](http://localhost:3000). API [health](http://localhost:8080/healthz) is process liveness; [readiness](http://localhost:8080/readyz) verifies PostgreSQL, Redis and the migrated schema. TURN health is localhost:8081/healthz.

Initialization preserves existing .env and generates separate random database, Redis and TURN secrets. Keep .env out of Git. Database/Redis URL passwords must be URL-safe (the generator uses hex). Changing the PostgreSQL password in .env does not change an already initialized database volume.

```sh
docker compose logs --tail 100
docker compose down
docker compose run --rm migrate
```

Down preserves database data. Do not add `--volumes` unless intentionally deleting it.

## Develop and verify

```sh
go mod download
go test ./...
go vet ./...
cd apps/web
npm ci
npm run typecheck
npm run lint
npm run build
npm run dev
```

Stop the Compose web service before using host dev on the same port. The Go backend uses DATABASE_URL, REDIS_URL, HTTP_ADDR, WEB_ORIGIN and APP_ENV; native Go does not automatically load .env. The root .env is consumed by Compose. Use Compose for the backend or explicitly export its connection URLs using localhost instead of service DNS names.

PowerShell `./scripts/dev.ps1 check` and Makefile targets provide equivalent checks. `smoke` requires the running stack. Integration checks and recorded results are in docs/verification.md.

## Layout

- apps/web — Next.js App Router, React and TypeScript.
- apps/server — Go application and separate embedded-Goose migration command.
- services/turn — Pion TURN process with UDP/TCP listeners, temporary auth, relay range and metrics.
- internal/ice — small shared credential/provider package.
- infra/docker — Go multi-stage Docker build.
- docs — architecture and explicit future milestone boundaries.
- scripts — cross-platform environment initialization, PowerShell commands and HTTP smoke checks.

## Decisions and next work

Compute remains disposable; PostgreSQL owns durable records and Redis will own distributed runtime state. Migrations run as a single deployment job before API startup. The initial migration creates pgvector and the minimal user/profile foundation; later tables ship with their working feature slices.

Application binaries run as non-root containers. Local web/API/database/cache ports bind to loopback. TURN has direct UDP/TCP access and a small bounded relay range; see [ICE/TURN](docs/webrtc/ice-turn.md) before LAN or VM use. The Compose file is for development, not a finished public deployment.

Milestone 1: a development-only two-browser room, typed WebSocket SDP/trickle-ICE signaling, camera/mic lifecycle, direct and forced-TURN tests, and an optional Pion diagnostic peer. Milestone 2 introduces Google verification and durable Go sessions before public access.

Read [architecture](docs/architecture/overview.md), [auth](docs/architecture/auth.md), [state](docs/architecture/state.md), [signaling](docs/signaling/protocol.md), and [matchmaking](docs/matchmaking/algorithm.md).

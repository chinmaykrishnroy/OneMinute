# Verification

## Milestone 0 (2026-09-05)

Executed locally on Windows with Docker Desktop Linux containers:
- Go 1.26.5 unit tests and go vet passed for backend configuration/readiness, temporary ICE credentials and TURN configuration.
- TURN relay range regression test passed (including rejection of requested ports outside the range).
- Next.js 16.3.4 production build, TypeScript 6.0.3 typecheck and ESLint passed.
- Docker Compose built and started PostgreSQL/pgvector, Redis, migration job, API, web and TURN. All long-running services passed health checks.
- HTTP smoke passed for API liveness/readiness, web health, TURN health and landing page.
- In-app browser rendered the landing page.
- Real integration tests passed: migration applied twice without duplicate changes, pgvector exact cosine query, user/profile constraints, Redis TTL, STUN binding and authenticated TURN allocations over UDP and TCP.
- TURN was rebuilt with the additional range guard and returned healthy.

Run `./scripts/integration.ps1` against the running development stack. It uses temporary keys and rolled-back SQL writes, never FLUSHDB or database deletion. Go integration tests require explicit environment variables and are excluded from normal unit tests.

ESLint is pinned to 9.39.5 because the current Next.js React/import plugins fail with ESLint 10; upgrade the plugins together when compatible. This is a development dependency. Dependency installation audit reported no vulnerabilities.

Not claimed: real camera/microphone, browser-to-browser media, relay data transfer through external NATs, Google authentication, public deployment, distributed matchmaking, or CI execution. Those require the subsequent milestones.

## Milestone 1 and remote verification (2026-09-05)

All testing moved to llm-04 after the owner requested it. Local OneMinute containers and the Compose network were removed; the durable local development volume was preserved. No older OneMinute deployment, Docker containers or volumes existed on llm-04 in the inspected locations.

Executed on llm-04:
- Full Docker build/start: PostgreSQL/pgvector, Redis, migration job, Go server, Next.js and independent Pion TURN. Every long-running service healthy.
- Go race detector across all packages and go vet: passed.
- Real PostgreSQL/Redis integration: migration re-runs, schema constraints, cosine query and Redis TTL: passed.
- Atomic room claims with 20 contenders: exactly two slots claimed.
- Two Go HTTP instances sharing Redis: signaling delivered across instances; third participant, forged match and hostile HTTP/WebSocket origins rejected.
- STUN binding and authenticated TURN allocations over UDP and TCP: passed.
- Pion end-to-end peers through the running Go WebSocket API: bidirectional synthetic Opus RTP and DataChannels, normal ICE and forced relay: passed. Forced relay requires both selected candidate pairs to be relay-to-relay. Both peers exit during cleanup.
- Frontend clean install, TypeScript, lint and production build: passed in a remote container.
- Remote HTTP readiness, web/TURN health, OneMinute HTML title and /lab route: passed.

The real transport tests exposed Docker bridge hairpin failures at the advertised TURN address. The Linux-only compose.remote.yaml places TURN on the host network and binds its health/metrics listener to 127.0.0.1. Re-running the failing transport tests and full regression then passed.

Not yet verified: browser-to-browser video playback, physical camera/microphone permissions, browser control/cleanup interactions, Pion-to-browser interoperability, or public internet NAT traversal. Browser automation was rejected by automatic approval review due to the account usage limit. It was not bypassed. The networking implementation is committed as a tested development slice; Milestone 1 browser acceptance remains open.

Remote commands and access constraints: docs/remote-development.md.

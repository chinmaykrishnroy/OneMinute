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

The initial browser attempt was blocked by the account usage limit. The later retry below succeeded. Physical camera/microphone permissions, Pion-to-browser interoperability and public internet NAT traversal remain unverified.

Remote commands and access constraints: docs/remote-development.md.

## Browser retry and UI direction (2026-09-05)

The in-app browser loaded the application from llm-04 through SSH forwards for ports 3000 and 8080. No application server, build or test runner ran on the workstation; browser media capture/playback ran in the browser as intended.

- Two browser peers joined a fresh room using the synthetic test pattern and silent audio source.
- Both reported connected, host-to-host UDP candidate pairs and increasing inbound media counters.
- Both remote videos played at 640x360, readyState 4 and paused false. This verifies synthetic video playback, not audible microphone capture.
- DataChannel messages were delivered in both directions.
- Mute/unmute and camera off/on controls toggled successfully.
- Leave ended both peers, emptied chat, reset media counters and disabled call/chat controls. Video elements returned to zero dimensions, readyState 0 and paused true.
- Both peers successfully joined a second fresh room and reconnected.
- Neither test tab reported browser console errors.

Browser relay testing is still pending: the remote configuration advertises private 10.160.3.131. The VM metadata reports public 34.100.213.131, but its firewall/public relay reachability has not been verified or changed. Earlier Pion forced-relay integration passed on llm-04; that is not a substitute for browser testing across public networks.

Cloudflare dashboard routes have been documented, not created. The existing connector was found active. See cloudflare-testing.md for Access protection, same-origin web/API routing and separate TURN requirements. The approved UI direction is saved in AGENTS.md and docs/design/ui-direction.md.

The shared UI stylesheet was updated to the approved visual direction. A clean frontend install, TypeScript check, lint and production build passed on llm-04. The rebuilt remote web service passed health checks, and the refreshed lab was visually inspected in the desktop browser.

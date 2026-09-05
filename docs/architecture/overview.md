# Architecture
Status: Milestones 0–1 foundation and development networking lab. Later features below are decisions, not claims of implementation.

One monorepo, one Go module (`example.com/encounter`), independently built binaries. The module is a neutral local import namespace; no repository is required at that URL. Branding lives in `apps/web/lib/brand.ts`.

- `apps/web`: Next.js App Router / React / TypeScript. Server Components render the shell; future media, signaling and timers belong in Client Components.
- `apps/server`: Go modular monolith using net/http, pgx and go-redis. Health/readiness, migration infrastructure and an explicitly gated Redis-backed signaling room.
- `services/turn`: independent Pion TURN UDP/TCP server. This process alone relays media when direct connectivity fails.
- `internal/ice`: provider-neutral temporary ICE configuration and shared-secret credential support.
- PostgreSQL 17 + pgvector: durable records. Redis 8: disposable live coordination.
- `migrate`: one deployment job using embedded Goose SQL migrations. Do not run concurrent migration jobs.

No SFU, Kubernetes, recording, or persistent chat. Pion WebRTC diagnostics are a separate optional executable at services/rtc-probe; normal calls never use a Go PeerConnection.

Local Compose binds web/API/database/cache to loopback. TURN listeners and relay ports are direct network bindings, with credentials required for relay allocations. This is a development stack, not a completed public deployment.

Next steps follow the requested milestones: networking proof, Google identity, distributed queue, product loop, durable social state, embeddings, hardening.

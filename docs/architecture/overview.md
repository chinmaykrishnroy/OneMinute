# Architecture
Status: Milestones 0–3 foundation, networking, identity and distributed discovery. Later features below are architecture decisions, not claims of implementation.

One monorepo, one Go module (`example.com/encounter`), independently built binaries. The module is a neutral local import namespace; no repository is required at that URL. Branding lives in `apps/web/lib/brand.ts`.

- `apps/web`: Next.js App Router / React / TypeScript. Server Components render the shell; future media, signaling and timers belong in Client Components.
- `apps/server`: Go modular monolith using net/http, pgx and go-redis. It owns application identity, health/readiness, migrations and the explicitly gated Redis-backed signaling lab.
- `services/turn`: independent Pion TURN UDP/TCP server. This process alone relays media when direct connectivity fails.
- `internal/ice`: provider-neutral temporary ICE configuration and shared-secret credential support.
- PostgreSQL 17 + pgvector: durable records. Redis 8: disposable live coordination.
- `migrate`: one deployment job using embedded Goose SQL migrations. Do not run concurrent migration jobs.

Normal one-to-one encounter and connection-call media stays browser-to-browser. Pion WebRTC diagnostics are a separate optional executable at services/rtc-probe; normal calls never use a Go PeerConnection. There is no call recording or public social feed.

Local Compose binds web/API/database/cache to loopback. TURN listeners and relay ports are direct network bindings, with credentials required for relay allocations. This is a development stack, not a completed public deployment.

The product loop is `DISCOVER → TALK → EXTEND → CONNECT → KEEP`. Discovery stays conversation-first: minimal identity and useful shared context before speaking, richer profiles after mutual Connect. Extend and Connect are independent private mutual-vote mechanisms. Extend changes only the current encounter; Connect creates the durable social edge.

The remaining sequence is distributed discovery, authoritative encounter lifecycle, durable social graph, persistent communication, intelligent discovery and hardening. See [roadmap](roadmap.md). MinIO will provide private S3-compatible object bytes behind a provider-neutral storage boundary; PostgreSQL stores metadata and references, never media blobs.

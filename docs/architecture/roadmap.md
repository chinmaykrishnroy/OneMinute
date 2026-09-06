# Product roadmap

The product loop is `DISCOVER → TALK → EXTEND → CONNECT → KEEP`. Milestones are vertical acceptance boundaries; later concepts listed here are architecture commitments, not implemented behavior.

## Completed foundation

### Milestone 0 — Foundation

Monorepo, Go and Next.js services, PostgreSQL/pgvector, Redis, migrations, health checks and container verification.

### Milestone 1 — Networking

Public same-origin WebSocket signaling, browser-to-browser WebRTC, independent Pion TURN, temporary credentials, cross-instance routing and direct/forced-relay acceptance tests.

### Milestone 2 — Identity

Google ID-token verification, stable provider identity, secure hashed application sessions, basic profile identity, returning-session behavior and logout/revocation.

## Completed discovery

### Milestone 3 — Discovery

Authenticated presence and sockets; queue membership; current session intent; language and session-scoped structured-interest compatibility; minimal durable block enforcement; recent-pair filtering; candidate selection; atomic two-user Redis claims; user-to-match mappings; finite match state; cross-instance routing; safe disconnect behavior. Match data must accommodate later lifecycle and private votes without implementing them now. Milestone 5 adds the user-facing block workflow and durable profile interests.

## Completed encounter

### Milestone 4 — Encounter

Authoritative 60-second lifecycle, server timestamps, responsive square-video encounter layout, local preview mirror/device settings, Next/Skip, private mutual Extend, RTCDataChannel temporary chat, 30–60 second reconnect grace, cleanup/requeue and atomic expiry/extension handling. Advanced camera effects remain an optional later enhancement behind a processed-track boundary.

## Planned product slices

### Milestone 5 — Social graph

Richer profiles and interests, block/report flows, reputation foundation, private mutual Connect, durable encounter-backed connections, connection list and removal/block interactions.

### Milestone 5.5 — Persistent communication

Durable direct messages, realtime delivery, private S3-compatible media storage initially backed by MinIO, presigned transfer, metadata and attachments, plus browser-to-browser audio/video calls between connected users using existing signaling/TURN infrastructure.

### Milestone 6 — Intelligent discovery

Qwen3 embeddings regenerated when relevant profile data changes, pgvector exact cosine ranking first, HNSW only after measurement, shared-context hints and an optional single icebreaker. Queue joins do not invoke embedding inference.

### Milestone 7 — Hardening

Moderation, abuse prevention, rate limits, upload validation and quotas, reports and bans, media cleanup, observability, security review, TURN hardening and production readiness.

## Product boundaries

Dating is an optional explicit intent and requires compatible opt-in from both users. Discovery is conversation-first and avoids full pre-call profiles or endless swipe/skip mechanics. Quality controls may later discourage instant-skip farming without imposing an unmeasured hard limit now. There is no public feed, follower count, likes, reels, stories system or call recording in this roadmap.

Private encounter “Moments” and privacy-safe share cards are possible later additions. They may retain permitted metadata such as date, duration and shared interests, never the call itself; identifiable sharing requires explicit consent.

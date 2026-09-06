# State ownership
Implemented:
- PostgreSQL: versioned schema, users, profiles with normalized interests/languages, Google external identities, hashed revocable sessions, blocks, encounter outcomes, reports and active/ended connections.
- Redis: development rooms (10-minute TTL), presence (40-second TTL), atomic two-slot claims, room-scoped authorization, transient Pub/Sub delivery and room-creation rate counters.
- Redis discovery: authenticated presence (40-second TTL), session preference queue, atomic two-user claims, user-to-match mappings, finite match state, recent-pair preference with sparse-queue fallback, disconnect markers and per-user cross-instance Pub/Sub routing.
- PostgreSQL social graph: directed blocks used by discovery, relationship-authorized reports, and canonical encounter-backed connections created only after mutual Connect.
- Process memory: connection pools, configuration, logs and TURN transport allocations only.

Redis encounter state includes authoritative match deadlines, separate private Extend and Connect votes, and a bounded reconnect-grace deadline. The active socket terminates on one instance; its owner is not the authority for domain state.

Pair claims and extension/expiry transitions use Lua atomically. No process-local mutex can arbitrate users across replicas. Redis loss ends live calls and requires requeue; PostgreSQL retains durable profiles and sessions. Redis uses noeviction so coordination keys are not silently evicted under memory pressure; application commands must handle OOM failures.

Persistent match outcomes, mutual connections, blocks and reports are in PostgreSQL. Conversations, messages and media metadata join them in later slices. Reliable delivery of durable terminal events needs an outbox/stream and retries before public release. Pub/Sub alone is suitable for transient SDP/ICE, presence, typing and delivery notifications, not durable outcomes or messages.

MinIO owns object bytes only and remains private by default. Application code uses an S3-compatible object-storage boundary so deployment can move to S3, R2 or another compatible provider. Process memory may own connection pools, configuration, local sockets and transport allocations, never authoritative distributed domain state.

Extend and Connect use separate Redis keys and transitions. A one-sided vote is private and expires with the encounter. Only a mutual Connect result is persisted, and it must reference the encounter that authorized it. Blocking supersedes a connection and every live/discovery eligibility check.

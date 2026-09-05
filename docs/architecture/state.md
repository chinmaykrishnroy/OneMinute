# State ownership
Implemented:
- PostgreSQL: versioned schema, users, profiles, Google external identities and hashed, revocable application sessions.
- Redis: development rooms (10-minute TTL), presence (40-second TTL), atomic two-slot claims, room-scoped authorization, transient Pub/Sub delivery and room-creation rate counters.
- Process memory: connection pools, configuration, logs and TURN transport allocations only.

Planned live state: expiring Redis presence, queue membership, current session intent, user-to-match mappings, match deadlines, Extend votes, Connect votes while an encounter is active, reconnect grace, recent-pair exclusions and socket/signaling routing. The active socket terminates on one instance; its owner cannot be the authority for domain state.

Pair claims and extension/expiry transitions use Lua atomically. No process-local mutex can arbitrate users across replicas. Redis loss ends live calls and requires requeue; PostgreSQL retains durable profiles and sessions. Redis uses noeviction so coordination keys are not silently evicted under memory pressure; application commands must handle OOM failures.

Persistent match outcomes, mutual connections, blocks, reports, conversations, messages and media metadata belong in PostgreSQL. Reliable persistence of terminal events needs an outbox/stream and retries before public release. Pub/Sub alone is suitable for transient SDP/ICE, presence, typing and delivery notifications, not durable outcomes or messages.

MinIO owns object bytes only and remains private by default. Application code uses an S3-compatible object-storage boundary so deployment can move to S3, R2 or another compatible provider. Process memory may own connection pools, configuration, local sockets and transport allocations, never authoritative distributed domain state.

Extend and Connect use separate Redis keys and transitions. A one-sided vote is private and expires with the encounter. Only a mutual Connect result is persisted, and it must reference the encounter that authorized it. Blocking supersedes a connection and every live/discovery eligibility check.

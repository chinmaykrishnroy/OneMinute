# State ownership
Implemented:
- PostgreSQL: versioned schema, users and profiles as the minimal durable foundation.
- Redis: connected dependency with readiness probing; no domain state exists yet.
- Process memory: connection pools, configuration, logs and TURN transport allocations only.

Planned live state: expiring Redis presence, queue membership, user-to-match mappings, match deadlines, extension votes, recent-pair exclusions and signaling delivery via Pub/Sub. The active socket terminates on one instance; its owner cannot be the authority for domain state.

Pair claims and extension/expiry transitions use Lua atomically. No process-local mutex can arbitrate users across replicas. Redis loss ends live calls and requires requeue; PostgreSQL retains durable profiles and sessions. Redis uses noeviction so coordination keys are not silently evicted under memory pressure; application commands must handle OOM failures.

Persistent match outcomes and reports must be idempotent. Reliable persistence of terminal events needs an outbox/stream and retries before public release. Pub/Sub alone is suitable for transient SDP/ICE, not durable outcomes.

# Schema roadmap

Status: migrations ship with the feature slice that owns them. Names below are conceptual until their milestone migration is reviewed.

## Implemented

- Milestone 0: `users`, `profiles`, pgvector extension and migration metadata.
- Milestone 2: `external_identities` keyed by provider plus stable provider subject; `sessions` with hashed secrets, expiry, last-seen and revocation.

## Milestone 3 — Discovery foundation

Redis owns presence, queue entries, current session intent/language/interests, match claims and user-to-match mappings. PostgreSQL gains only the minimal directed `blocks(blocker_user_id, blocked_user_id, created_at)` record needed for real candidate exclusion. One unordered pair cannot bypass a block in either direction. Public block creation and management remain part of Milestone 5.

Match state is initially live and finite in Redis. Its identifiers and participant ordering must be suitable for later durable encounter outcomes and mutual Connect creation. Session-scoped interests are not written to profile tables in this milestone.

## Milestone 4 — Encounter outcomes

Persist only durable, privacy-safe encounter outcome metadata needed by product and abuse rules. Live deadlines, reconnect grace and private Extend votes stay in Redis. Calls and temporary DataChannel messages are never stored.

## Milestone 5 — Social graph

Add normalized durable profile interests/preferences, reports/reputation inputs and encounter-backed `connections`. Store each unordered connection pair canonically and enforce at most one active relationship for that pair. Connection creation must reference a valid encounter with mutual Connect; arbitrary out-of-encounter friend requests are invalid. Removal and blocking are distinct, and a block supersedes connection visibility and authorization.

Private Connect votes stay in Redis while the encounter is active. Only their mutual outcome becomes durable. One-sided votes expire without notification or durable social edge.

## Milestone 5.5 — Persistent communication

Add one durable conversation per active connection as policy permits, ordered messages, optional receipts and `media_objects` metadata. Messages are authorized through the connection and may reference ready media metadata. Object bytes remain in private S3-compatible storage and never in PostgreSQL.

## Milestone 6 — Intelligent discovery

Store normalized source text, model/version and 1024-dimensional Qwen3 embeddings only when relevant durable profile data changes. Start with exact cosine distance. Add an HNSW index only after query measurements justify its write/storage cost.

## Migration rules

Use UUID primary keys, explicit foreign keys and deletion behavior, timestamps, bounded text and database constraints for invariants. Durable terminal writes must be idempotent. Prefer canonical unordered-pair columns with a check constraint plus a partial unique index for active relationships. Do not create speculative tables before the feature and its authorization tests exist.

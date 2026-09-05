# Matchmaking
Status: architecture for Milestones 3 and 6; no production matching or embeddings are implemented yet.

Each queue entry captures the authenticated user's current session intent, language choices and structured interests. Session intent has priority over profile-level allowed intents. A dating candidate is eligible only when both users explicitly selected compatible dating intent; mismatched private intent is never disclosed.

For Milestone 3, interests are a bounded, validated session selection carried in Redis; durable profile interests arrive with richer profiles in Milestone 5. Block filtering requires a minimal durable block record and repository query in Milestone 3 so the eligibility rule is real and testable. The user-facing block/report workflow remains Milestone 5.

Filter first: authenticated live user, unexpired queue entry, compatible session intent, compatible language, no active match, active account, no block in either direction, no recent pair and applicable safety constraints. Revalidate availability, block state and safety at the atomic claim boundary. Simple structured interests are enough for Milestone 3 and can produce shared-interest context such as “You both like Music and AI.”

Rank eligible candidates with configuration-held weights:
`0.55 * semanticSimilarity + 0.15 * queueWaitBonus + 0.10 * reputationScore + 0.20 * randomness`.
Normalize each term to [0,1], clamp cosine similarity appropriately, bound the wait bonus, and use unbiased randomness. New users start with neutral reputation. Reports alone must not cause large score penalties.

Claim both users in one Redis Lua operation: confirm presence/queue/reservations, reject existing matches, remove queue entries, create finite-TTL match state, map both users to it. Coordinate block changes with the claim path so a stale filtered candidate cannot bypass a new block.

Match state must have room for authoritative lifecycle status and a reconnect grace marker, while keeping later Extend and Connect votes in separate keys/state machines. A client never selects a destination user or proves match membership itself. Milestone 3 handles safe disconnect and routing; the 60-second clock, reconnect window and voting transitions arrive in later milestones.

Later: normalize profile interests, generate Qwen3-Embedding-0.6B vectors on relevant profile changes, persist model/version/text/vector(1024) in pgvector. Do not invoke inference on queue join. Start with exact cosine distance; HNSW can later accelerate the same repository query.

The optional later icebreaker receives bounded shared context and returns one short starter. It is not part of matching authority and is not required for Milestone 3.

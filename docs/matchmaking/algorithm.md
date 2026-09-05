# Matchmaking
Status: architecture for Milestones 3–6; no matching or embeddings are implemented yet.

Filter first: live presence, unexpired queue entry, compatible language, no active match, active account, no block in either direction, no recent pair. Revalidate availability and safety at the atomic claim boundary.

Rank eligible candidates with configuration-held weights:
`0.55 * semanticSimilarity + 0.15 * queueWaitBonus + 0.10 * reputationScore + 0.20 * randomness`.
Normalize each term to [0,1], clamp cosine similarity appropriately, bound the wait bonus, and use unbiased randomness. New users start with neutral reputation. Reports alone must not cause large score penalties.

Claim both users in one Redis Lua operation: confirm presence/queue/reservations, reject existing matches, remove queue entries, create finite-TTL match state, map both users to it. Coordinate block changes with the claim path so a stale filtered candidate cannot bypass a new block.

Later: normalize profile interests, generate Qwen3-Embedding-0.6B vectors on relevant profile changes, persist model/version/text/vector(1024) in pgvector. Do not invoke inference on queue join. Start with exact cosine distance; HNSW can later accelerate the same repository query.

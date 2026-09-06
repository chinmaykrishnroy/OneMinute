# Signaling protocol
Status: the isolated development lab, authenticated discovery sockets, authoritative encounter lifecycle, Extend and Connect events are implemented. See packages/protocol/README.md for envelope constraints.

Envelope:
```json
{"version":1,"type":"webrtc.ice","requestId":"opaque-id","matchId":"uuid","payload":{}}
```

Client events: presence.heartbeat, queue.join, queue.leave, webrtc.offer, webrtc.answer, webrtc.ice, match.ready, match.extend, match.connect, match.skip, match.leave.
Server events: connection.ready, queue.joined, match.found, match.started, webrtc.offer, webrtc.answer, webrtc.ice, peer.disconnected, peer.reconnected, match.extended, connection.created, match.ended, error.

Validate version, event type, field lengths, payload shape and match membership on the server. Never accept an arbitrary destination user UUID. Bound frames and per-connection/per-user rates. Correlate errors with requestId; never log SDP or ICE bodies.

Redis Pub/Sub routes transient events across instances. A 45-second reconnect grace handles accidental network changes. Reconnect revalidates the Go session, live match membership, match status and grace deadline; the client cannot restore itself by asserting a match ID. Redis match state lets a reconnect recover the current lifecycle snapshot after missed Pub/Sub events.

Both clients use authoritative startedAt/expiresAt timestamps; the backend resolves Extend versus expiry atomically using server time. One-sided Extend or Connect votes remain private. Extend removes the current encounter deadline only after both votes and creates no durable relationship. Connect remains independent and creates a durable encounter-backed connection only after both votes.

Temporary encounter text uses RTCDataChannel and is not stored. Later persistent connection messages use authenticated WebSocket/HTTP delivery with PostgreSQL as the source of truth; they do not reuse the DataChannel protocol.

# Signaling protocol
Status: the development lab implements room.join, presence.heartbeat, webrtc.offer/answer/ice and match.leave; server events include connection.ready, match.found, match.ended and error. The queue and sixty-second lifecycle below remain planned. See packages/protocol/README.md for the implemented envelope constraints.

Envelope:
```json
{"version":1,"type":"webrtc.ice","requestId":"opaque-id","matchId":"uuid","payload":{}}
```

Client events: presence.heartbeat, queue.join, queue.leave, webrtc.offer, webrtc.answer, webrtc.ice, match.ready, match.extend, match.skip, match.leave.
Server events: connection.ready, queue.joined, match.found, match.started, webrtc.offer, webrtc.answer, webrtc.ice, peer.disconnected, match.extended, match.ended, error.

Validate version, event type, field lengths, payload shape and match membership on the server. Never accept an arbitrary destination user UUID. Bound frames and per-connection/per-user rates. Correlate errors with requestId; never log SDP or ICE bodies.

Redis Pub/Sub routes transient events across instances. Reconnect deliberately ends/revalidates the old session rather than silently trusting a client's match assertion. Durable match state must let a reconnect detect missed lifecycle events.

Both clients use authoritative startedAt/expiresAt timestamps; the backend resolves extension versus expiry atomically using server time. Do not reveal the other user's private extend vote before mutual extension.

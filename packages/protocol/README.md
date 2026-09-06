# Protocol artifacts
The initial networking protocol is implemented in Go at apps/server/internal/signaling/protocol.go and mirrored in TypeScript at apps/web/lib/signaling/protocol.ts. The Go decoder is the security boundary; the client types improve implementation safety.

Version 1 lab client events: room.join, presence.heartbeat, webrtc.offer, webrtc.answer, webrtc.ice, match.leave.
Version 1 lab server events: connection.ready, match.found, webrtc.offer, webrtc.answer, webrtc.ice, match.ended, error.

room.join is a temporary development-only capability flow. The application endpoint `/v1/discovery/ws` authenticates the Go session and exact Origin before accepting a socket. Discovery and encounter client events are `queue.join`, `queue.leave`, `presence.heartbeat`, `webrtc.offer`, `webrtc.answer`, `webrtc.ice`, `match.ready`, `match.extend`, `match.connect`, `match.skip` and `match.leave`. Server events are `connection.ready`, `queue.joined`, `queue.left`, `match.found`, `match.started`, `peer.disconnected`, `peer.reconnected`, relayed WebRTC events, `match.extend_pending`, `match.extended`, `match.connect_pending`, `connection.created`, `match.ended` and `error`.

`queue.join` carries a bounded current intent, one to three language tags and up to eight allowlisted session interests. It never carries a destination user ID. The server derives identity from the session cookie, and every match-scoped event is authorized against Redis user-to-match state and the active socket connection. Extend and Connect are separate private mutual-vote transitions; only mutual Connect creates an encounter-backed PostgreSQL connection.

All envelopes carry version/type/payload, optional requestId, and matchId for active-room signals. Payloads are objects; null, unknown top-level/payload fields, unsupported versions/events and oversized messages are rejected by Go. SDP is limited to 60,000 characters; ICE candidates to 4,096; frames to 64 KiB. Chat uses the RTCDataChannel and is absent from this protocol.

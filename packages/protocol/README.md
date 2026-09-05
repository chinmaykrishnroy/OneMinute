# Protocol artifacts
The initial networking protocol is implemented in Go at apps/server/internal/signaling/protocol.go and mirrored in TypeScript at apps/web/lib/signaling/protocol.ts. The Go decoder is the security boundary; the client types improve implementation safety.

Version 1 lab client events: room.join, presence.heartbeat, webrtc.offer, webrtc.answer, webrtc.ice, match.leave.
Version 1 lab server events: connection.ready, match.found, webrtc.offer, webrtc.answer, webrtc.ice, match.ended, error.

room.join is a temporary development-only capability flow. Queue and identity events arrive in later milestones. Do not expose arbitrary destination user IDs.

All envelopes carry version/type/payload, optional requestId, and matchId for active-room signals. Payloads are objects; null, unknown top-level/payload fields, unsupported versions/events and oversized messages are rejected by Go. SDP is limited to 60,000 characters; ICE candidates to 4,096; frames to 64 KiB. Chat uses the RTCDataChannel and is absent from this protocol.

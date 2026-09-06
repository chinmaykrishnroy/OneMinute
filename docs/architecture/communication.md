# Persistent communication

The communication core of Milestone 5.5 keeps connection messages in PostgreSQL and call/signaling runtime state in Redis. It is independent of temporary encounter RTCDataChannel chat and does not change the private Extend/Connect votes.

## Messaging and notifications

`GET /v1/conversations` lists active, unblocked connections ordered by recent activity, with a bounded last-message preview and unread count. `GET /v1/connections/{id}/messages` returns the newest 60 messages; `before` pages backward. `POST` accepts a UUID `clientId` and 1–4000 Unicode characters. The sender/client pair is unique, so retrying a lost response returns the original message. A transaction locks the active connection through commit and saves the recipient's notification alongside the message. Removal and blocks prevent subsequent reads, writes, typing and calls.

`POST /v1/connections/{id}/receipt` advances monotonic delivered/read cursors for incoming messages only. Duplicate cursors do not emit another event. Disabling read receipts suppresses future shared read advances; it cannot retract receipts already shared. Reading a conversation still clears its private notification state. Typing is transient, respects the sender setting and expires in the UI after four seconds.

`GET /v1/events/ws` requires the application session and exact web Origin. It returns the current identity, settings and temporary TURN configuration, then delivers Redis events across Go instances. The socket revalidates its session every 20 seconds. The inbox reloads durable history after interruptions and polls as a missed-event fallback. User mutations have a shared Redis limit of 360/minute, and call invitations have a separate 6/minute limit. Broader abuse policy and capacity controls remain Milestone 7 work.

Notifications persist for messages, call attempts and new mutual connections. The header drawer lists recent events and can mark them read. Native browser alerts require an explicit browser permission gesture and are available while the page remains open. This implementation does not claim service-worker Web Push, closed-browser incoming calls or native iOS/Android push delivery.

## Connected calls

`POST /v1/connections/{id}/calls` atomically reserves both participants and creates a 45-second ringing state. Only the callee can accept. Acceptance renews the busy leases; active heartbeats renew the 75-second call lease. Ending/declining clears the call and both owned leases. A block/removal ends the call on the next validated action or heartbeat. Acceptance is also broadcast to the callee's other tabs so they cannot later cancel an already answered call.

`POST /v1/calls/{id}` accepts bounded offer, answer, ICE, media state and lifecycle messages only from call participants. The caller offers and callee answers. Browser-to-browser media uses authenticated TURN credentials from the existing provider; Go routes signaling and never records or mixes media. Turning the camera on/off replaces the already-negotiated video sender. Microphone mute controls the local audio track. Camera changes replace the processed track, with selfie mirroring applied to the actual transmitted frames and local preview.

The remote video fills the available call surface using full-frame containment. The local preview follows camera aspect ratio and can be dragged or moved with arrow keys within bounds. A native modal keeps keyboard focus within the call. Closing, leaving the page, lost signaling and bounded connection timeouts release local media. Device capture requests ideal 1080p without a minimum; WebRTC congestion control chooses the effective sending quality.

## Account setup and settings

New/incomplete accounts enter `/onboarding`, which collects display name, conversation intent, at least one language and optional interests before entering Discover. The UI explains that You is available for later edits. Settings persist in `user_settings`: System (default), Light or Dark; browser notifications; typing; and read receipts. System mode follows live device appearance changes.

Application builds and tests run only on `ssh oneminute` using `compose.test.yaml`. `npm run typecheck` includes a UTF-8/replacement-character check to prevent the reported broken-label regression. See `docs/verification.md` for executed checks and the outstanding physical-device acceptance matrix.

# Networking lab (Milestone 1)

Enable `RTC_LAB_ENABLED=true` in the local .env, run `docker compose up --build --detach --wait`, and open http://localhost:3000/lab in two tabs.

1. Create a room in tab A.
2. Choose Camera and microphone (requires permission), or Test pattern (synthetic video and a silent audio track).
3. Join tab A, copy its room code into tab B, select a source and join B.
4. Check remote video, selected ICE candidate types and increasing received-media bytes.
5. Send a DataChannel message, toggle mute/camera, and leave. Both tabs must clear video/chat and close resources.
6. Create a fresh room, select Force TURN relay in both tabs and repeat. The selected pair should report relay candidates.

Room codes are random 128-bit development capabilities; anyone with the code can fill an available slot. There is no account identity in this lab. Rooms expire after ten minutes and presence after 40 seconds without a heartbeat. They cannot be reused after a peer leaves. The flag defaults off and Go refuses to start with the lab enabled in production. The web route also returns 404 in production.

The exact configured WEB_ORIGIN is required by HTTP and WebSocket entry points. API_PUBLIC_URL is the browser-reachable API origin (default localhost:8080). The backend issues ten-minute ICE credentials only after claiming a valid room slot. No permanent TURN secret reaches the browser.

For LAN/VM browser testing, configure HTTPS for the web/API, set both origins, and set TURN_PUBLIC_IP to the reachable LAN/public IPv4. Compose intentionally binds web/API to loopback; use a suitable local HTTPS reverse proxy or a deliberate deployment override. TURN is published directly.

## Optional Pion peer

```sh
go run ./services/rtc-probe
# Or join an existing browser room:
go run ./services/rtc-probe -room <room-code> -relay -test-audio
```

The probe creates a room when -room is omitted. Join that code from the web lab. It logs candidate types, gathering/connection state, selected candidate pair, incoming audio/video codecs and DataChannel delivery. It receives browser media and exchanges diagnostic chat; it does not capture a camera, record media, or serve as the normal media path.

The optional -test-audio flag sends synthetic Opus silence for RTP checks. Two probes exchange greetings and bounded acknowledgments, without an echo loop. The command exits on room end, interrupt, or a ten-minute lifetime limit.

## Known boundaries

No account login, random queue, sixty-second expiry, extension, skip/requeue or persistent social state yet. These belong to Milestones 2–6. The lab deliberately ends on signaling loss instead of silently attempting session resumption. Transient packet loss can recover at the PeerConnection level; ICE failure requires a fresh room.

Redis owns room membership and signaling authorization. Lua enforces two slots and sender membership. Pub/Sub routes across API instances. Only WebSocket transport resources are local to an instance. Pub/Sub can lose transient messages during a Redis outage; this lab is a network proof, not the completed recovery protocol.

Media and chat remain peer-to-peer unless TURN is required. The browser buffers early trickle candidates until remote SDP is set and closes old streams, tracks, timers, channels, sockets and peer connections on leave/unmount/error. Chat is bounded to 50 transient messages of 2,000 characters each.

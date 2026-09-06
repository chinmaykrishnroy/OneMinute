# Verification

## Milestone 0 (2026-09-05)

Executed locally on Windows with Docker Desktop Linux containers:
- Go 1.26.5 unit tests and go vet passed for backend configuration/readiness, temporary ICE credentials and TURN configuration.
- TURN relay range regression test passed (including rejection of requested ports outside the range).
- Next.js 16.3.4 production build, TypeScript 6.0.3 typecheck and ESLint passed.
- Docker Compose built and started PostgreSQL/pgvector, Redis, migration job, API, web and TURN. All long-running services passed health checks.
- HTTP smoke passed for API liveness/readiness, web health, TURN health and landing page.
- In-app browser rendered the landing page.
- Real integration tests passed: migration applied twice without duplicate changes, pgvector exact cosine query, user/profile constraints, Redis TTL, STUN binding and authenticated TURN allocations over UDP and TCP.
- TURN was rebuilt with the additional range guard and returned healthy.

Run `./scripts/integration.ps1` against the running development stack. It uses temporary keys and rolled-back SQL writes, never FLUSHDB or database deletion. Go integration tests require explicit environment variables and are excluded from normal unit tests.

ESLint is pinned to 9.39.5 because the current Next.js React/import plugins fail with ESLint 10; upgrade the plugins together when compatible. This is a development dependency. Dependency installation audit reported no vulnerabilities.

Not claimed: real camera/microphone, browser-to-browser media, relay data transfer through external NATs, Google authentication, public deployment, distributed matchmaking, or CI execution. Those require the subsequent milestones.

## Milestone 1 and remote verification (2026-09-05)

All testing moved to llm-04 after the owner requested it. Local OneMinute containers and the Compose network were removed; the durable local development volume was preserved. No older OneMinute deployment, Docker containers or volumes existed on llm-04 in the inspected locations.

Executed on llm-04:
- Full Docker build/start: PostgreSQL/pgvector, Redis, migration job, Go server, Next.js and independent Pion TURN. Every long-running service healthy.
- Go race detector across all packages and go vet: passed.
- Real PostgreSQL/Redis integration: migration re-runs, schema constraints, cosine query and Redis TTL: passed.
- Atomic room claims with 20 contenders: exactly two slots claimed.
- Two Go HTTP instances sharing Redis: signaling delivered across instances; third participant, forged match and hostile HTTP/WebSocket origins rejected.
- STUN binding and authenticated TURN allocations over UDP and TCP: passed.
- Pion end-to-end peers through the running Go WebSocket API: bidirectional synthetic Opus RTP and DataChannels, normal ICE and forced relay: passed. Forced relay requires both selected candidate pairs to be relay-to-relay. Both peers exit during cleanup.
- Frontend clean install, TypeScript, lint and production build: passed in a remote container.
- Remote HTTP readiness, web/TURN health, OneMinute HTML title and /lab route: passed.

The real transport tests exposed Docker bridge hairpin failures at the advertised TURN address. The Linux-only compose.remote.yaml places TURN on the host network and binds its health/metrics listener to 127.0.0.1. Re-running the failing transport tests and full regression then passed.

The initial browser attempt was blocked by the account usage limit. The later retry below succeeded. Physical camera/microphone permissions, Pion-to-browser interoperability and public internet NAT traversal remain unverified.

Remote commands and access constraints: docs/remote-development.md.

## Browser retry and UI direction (2026-09-05)

The in-app browser loaded the application from llm-04 through SSH forwards for ports 3000 and 8080. No application server, build or test runner ran on the workstation; browser media capture/playback ran in the browser as intended.

- Two browser peers joined a fresh room using the synthetic test pattern and silent audio source.
- Both reported connected, host-to-host UDP candidate pairs and increasing inbound media counters.
- Both remote videos played at 640x360, readyState 4 and paused false. This verifies synthetic video playback, not audible microphone capture.
- DataChannel messages were delivered in both directions.
- Mute/unmute and camera off/on controls toggled successfully.
- Leave ended both peers, emptied chat, reset media counters and disabled call/chat controls. Video elements returned to zero dimensions, readyState 0 and paused true.
- Both peers successfully joined a second fresh room and reconnected.
- Neither test tab reported browser console errors.

Browser relay testing is still pending: the remote configuration advertises private 10.160.3.131. The VM metadata reports public 34.100.213.131, but its firewall/public relay reachability has not been verified or changed. Earlier Pion forced-relay integration passed on llm-04; that is not a substitute for browser testing across public networks.

Cloudflare dashboard routes have been documented, not created. The existing connector was found active. See cloudflare-testing.md for Access protection, same-origin web/API routing and separate TURN requirements. The approved UI direction is saved in AGENTS.md and docs/design/ui-direction.md.

The shared UI stylesheet was updated to the approved visual direction. A clean frontend install, TypeScript check, lint and production build passed on llm-04. The rebuilt remote web service passed health checks, and the refreshed lab was visually inspected in the desktop browser.

## Milestone 1 acceptance on the dedicated host (2026-09-05)

Deployment moved to `ssh oneminute`, `/home/roy/OneMinute`. GitHub fetch works directly after the repository was made public. Fresh remote-only secrets were generated. The public endpoint is https://oneminute.prefect-sys.online/lab and TURN advertises 35.234.222.18.

Verified on the new machine:

- Full Docker deployment: PostgreSQL/pgvector, Redis, migration job, Go signaling, Next.js, Pion TURN and Caddy gateway. All six long-running services healthy; migration completed successfully.
- Caddy configuration validation and public HTTPS health passed. One Cloudflare route to HTTP 127.0.0.1:3000 serves both web and `/v1/*`, including public WebSocket signaling.
- Full Go race detector and vet passed. Real database/Redis integration, atomic two-slot room claims, cross-instance signaling and protocol rejection checks passed.
- STUN binding and authenticated TURN allocations passed over UDP and TCP, returning relay addresses within 49160–49180.
- Pion end-to-end tests passed in normal and forced-relay modes, with bidirectional synthetic Opus RTP, DataChannels and clean peer termination.
- Frontend clean install, TypeScript, lint and production build passed.

Verified from browsers using the public HTTPS hostname:

- Two generated-media browser peers connected with Force TURN relay enabled on both. Both selected `relay ↔ relay (udp)`, received increasing media bytes, and played 640x360 remote video with readyState 4 and paused false.
- Messages arrived in both directions. Mute/unmute and camera off/on controls toggled; Leave ended both peers, cleared chat/counters and returned video elements to readyState 0 and zero dimensions.
- A Pion peer running on the dedicated VM joined through the public HTTPS/WebSocket endpoint with relay required. Pion received browser VP8 video and Opus audio; the browser received synthetic Pion audio RTP and a greeting, sent a message, and received an acknowledgment. Pion reported relay-to-relay and exited successfully after browser Leave.
- A fresh normal-ICE browser call then connected host-to-host over UDP with increasing media counters and an open DataChannel. Both browser tabs reported no console errors.

Milestone 1 implementation and networking acceptance are complete. Device-specific camera/microphone permissions and audible physical microphone playback remain manual acceptance checks. Generated media exercises real RTP but does not verify physical hardware. The browser pair used one workstation; a Wi-Fi/cellular two-device matrix and restrictive-network TURN/TLS fallback are not claimed. The current fallback supports TURN/TCP 3478, not TURN/TLS 443.

After these checks, the old llm-04 database was confirmed to contain zero users. Its OneMinute containers, network, database volume, two Go test cache volumes, application images, test toolchain images, `/home/roy/OneMinute`, and the two known repository transfer archives were removed. No Docker containers or volumes remained. The unrelated llama.cpp image was preserved, and cloudflared and Tailscale remained active. Shared Docker build cache was not globally pruned because llm-04 also hosts unrelated LLM work.

## Milestone 2 identity and architecture checkpoint (2026-09-06)

Implemented and deployed on `oneminute`:

- Google Identity Services client UI with a server-issued nonce and backend ID-token validation for signature, issuer, audience, expiry and nonce.
- Stable Google subject mapping to application users, basic name/avatar identity, and an email-verification signal without storing or keying identity by email.
- Opaque 30-day application sessions in Secure, HttpOnly, SameSite=Lax host cookies. PostgreSQL stores only SHA-256 session-secret hashes with expiry, last-seen and revocation.
- Exact Origin checks on login and logout, bounded strict JSON, returning-session lookup, logout/revocation, and no browser-local token storage.
- Public same-origin routing for identity endpoints. The frontend build now receives `API_PUBLIC_URL`; this fixed a discovered mixed-content failure where the static page had compiled the localhost fallback.

Remote verification passed: full Go race suite and vet; authentication lifecycle integration with new user, hardened cookie, hash-only storage, returning identity and revoked-session rejection; existing PostgreSQL/Redis, signaling, TURN UDP/TCP and Pion normal/forced-relay integration; clean frontend install, typecheck, lint and production build. All deployed services were healthy. The public HTTPS page rendered the Google button with no page errors, and `/v1/auth/config` returned enabled configuration with a hardened nonce cookie.

Not yet claimed: completing the real Google account selector, confirming the resulting production Google ID token against the live backend, and visually confirming the signed-in avatar/returning session/logout in a user account. That action requires an account holder to choose an account in Google’s UI. The automated verifier-boundary integration covers the same application session lifecycle with a controlled signed-credential verifier.

The product architecture was updated before Milestone 3 to record the conversation-first product loop, optional intent model, separate Extend and Connect state machines, reconnect grace, durable connections and messaging, S3-compatible private media storage, schema sequencing and revised milestones. No Milestone 3 runtime feature was implemented in this checkpoint.

The owner subsequently supplied a screenshot of the authenticated public `/app/discover` route, confirming that a real Google-backed application session reached the protected discovery UI and displayed the session-ready state. Returning-session persistence and logout remain available in the implementation and automated lifecycle test; a separate manual report of those two actions has not been recorded.

## Milestone 3 discovery (2026-09-06)

Implemented and deployed on `oneminute`:

- Exact-Origin, application-session-authenticated WebSocket discovery endpoint with 64 KiB frame bounds and per-socket rate limiting.
- Redis presence heartbeats, session-scoped intent/language/interests, queue membership, stale-entry cleanup, finite match state, user-to-match mappings, recent-pair TTLs and per-user Pub/Sub routing.
- Candidate filtering for compatible current intent and language, mutual Dating opt-in, active accounts, directed blocks and recent pairs. Candidates rank by shared structured-interest count and queue age for this milestone.
- Atomic Lua claim of both live, queued socket connections. Every queue mutation, match leave and WebRTC signal is bound to the active connection ID and server-owned match membership.
- Cross-instance match and WebRTC signaling delivery, disconnect notification, and revalidated active-match recovery after reconnect.
- Protected `/app/discover` UI with current intent, language and bounded interest selection, shared-interest match context and a responsive soft-neobrutalist layout.

Remote verification passed: full Go race suite and vet; 20-way atomic claim contention; hostile-origin and unauthenticated WebSocket rejection; two Go HTTP instances sharing Redis; real PostgreSQL active-account/block checks; compatible match creation; shared-interest context; forged-match rejection; cross-instance offer delivery; disconnect/reconnect recovery; match leave; recent-pair avoidance; and blocked-pair exclusion. Existing auth, migration, signaling, TURN UDP/TCP and Pion normal/forced-relay integrations also passed. Frontend clean install, typecheck, lint and production build passed, and all deployed services were healthy.

The public authentication boundary redirected an unsigned `/app/discover` request back to the Google sign-in page without browser console errors. The owner’s authenticated screenshot exposed an interest-chip overflow issue; the deployed fix uses a shrinking fieldset and responsive 130-pixel-minimum grid so options wrap within the card on desktop, tablet and mobile.

Not yet claimed: a real two-account browser matchmaking session. The real integration suite exercises the same authenticated handlers, PostgreSQL/Redis state and cross-instance WebSocket flow with controlled test identities. Camera/video, the authoritative 60-second timer, Extend and complete reconnect grace are Milestone 4 work.

## Milestone 4 encounter (2026-09-06)

Implemented and deployed on `oneminute`:

- Redis owns the 60-second encounter deadline and emits terminal events from atomic Lua transitions. The UI renders the server timestamps; it does not decide when a match expires.
- Extend is a private per-user vote. One vote receives a private pending acknowledgment; only two votes atomically remove the deadline and notify both participants. A deadline comparison inside the same script prevents a late extension from reviving an expired encounter.
- Next atomically ends the current match and rejoins with the same session preferences after cleanup. Leave ends without requeueing. Recent-pair suppression remains active.
- A disconnected participant has a server-enforced 45-second reservation. Reconnect revalidates the session, live connection, match membership, and grace deadline. A shared sweeper safely handles encounter and disconnect deadlines across instances.
- The protected encounter screen creates the browser-to-browser media connection with authenticated ICE credentials and existing cross-instance signaling. Temporary chat uses an ordered RTCDataChannel and is not sent to or stored by the application server.
- The screen uses equal square video tiles on wide displays, stacked remote/local video on phones, a portrait-tablet stack, and a compact landscape layout. The local camera panel supports preview-only mirroring and live input-device replacement. Advanced processed-track effects remain future work.

Remote verification passed: the full Go race and vet suite; PostgreSQL/Redis integration; authoritative timestamps; private then mutual extension; skip cleanup; reconnect recovery and post-grace rejection; forged-match rejection; cross-instance signaling; concurrent expiry/extension behavior; TURN UDP/TCP; Pion normal and forced-relay media/DataChannels; and frontend typecheck, lint, and production build. The deployment rebuilt successfully with the remote/public Compose overlays. PostgreSQL, Redis, Go, Next.js, Pion TURN and Caddy are healthy; local readiness and both local and public gateway health endpoints pass.

Still requires manual acceptance with two real authenticated accounts and physical cameras: permission prompts, visible/audible device media, the exact responsive layout on representative phone/tablet/desktop hardware, a full natural expiry, mutual Extend, Next requeue, a network-drop reconnect within 45 seconds, and direct browser DataChannel chat. Automated WebRTC media/TURN coverage uses the existing lab and Pion tests.

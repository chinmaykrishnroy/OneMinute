# Verification

## Communication core and product UX (2026-09-06)

Implemented: dedicated first-time onboarding; individual tab titles; corrected UTF-8 labels and a build-time regression check; sticky headers and spaced mobile tabs; persisted System/Light/Dark appearance and notification/typing/read-receipt settings; connection-only PostgreSQL text messaging; retry-safe client message IDs; realtime delivery, typing and receipts; inbox previews/unread counts; durable connection/message/call notifications; and browser-to-browser connected calls with audio/video switching and a draggable, aspect-aware self-preview.

Executed on `ssh oneminute` with containerized verification: frontend clean install, UTF-8 guard, TypeScript, ESLint and production build (including onboarding/settings routes); Go race tests and vet; complete integration suite against PostgreSQL/Redis. New lifecycle coverage checks anonymous/hostile-origin/outsider rejection, settings persistence/defaults, message history and duplicate retry, incoming-only receipts, duplicate-receipt suppression, conversation preview/unread state, notification generation/read state, busy calls, caller/callee role enforcement, acceptance/offer/answer/media/heartbeat and block-driven cleanup. Existing discovery/social/TURN UDP/TCP and Pion direct/forced-relay media/DataChannel tests also pass.

These transport tests do not certify the new connected-call browser UI against two physical devices. Pending acceptance: audible/visible two-account calls, microphone/camera permissions, camera switching on Android/iOS, orientation changes and dock dragging, bandwidth degradation, background tab behavior and device/network interruption. Closed-app Web Push/native push and private message attachments remain unimplemented; browser alerts currently require the site to remain open and permission to be granted. Posts remains an explicit placeholder without a public feed.

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

### Mirrored media and sparse-queue fallback amendment

The camera path now draws the selected input into a canvas-backed track used by both the local preview and WebRTC sender. Selfie-style mirroring is enabled by default, and changing it updates both views. Recent pairs remain lower priority, but the matcher may claim one when no fresh compatible candidate is available; this allows two test accounts to meet repeatedly without waiting for the recent-pair TTL.

Remote automated verification covers the recent-pair fallback, the full Go race/vet and integration regression suite, TURN UDP/TCP, Pion normal/forced-relay media, and frontend typecheck, lint and production build. The updated deployment is healthy. Public checks return the generated SVG, PNG, ICO and web manifest with their expected content types, and `/app/discover` renders `Discover · OneMinute`. Visual equivalence of the local and remote physical-camera image, including a live mirror toggle, remains a two-device manual check.

### Encounter media recovery amendment

After a manual report that returning from Home notified the waiting peer without restoring video, both sides now rebuild their peer connection on `peer.reconnected`; the designated offerer renegotiates while an existing camera pipeline is reused. Camera acquisition requests ideal 1920×1080 with no minimum, and the sender selects balanced degradation so the browser's WebRTC congestion controller may reduce resolution and bitrate under constrained bandwidth.

Frontend typecheck, lint and production build passed on `oneminute`. A real two-account Home → Discover recovery and observing resolution changes under controlled bandwidth remain manual browser checks; the implementation does not claim exact resolution thresholds because the browser chooses them from live congestion signals and device capabilities.

## Milestone 5 social graph (2026-09-06)

Implemented and deployed on `oneminute`:

- Profile editing stores bounded display name, bio, country, normalized interests and language tags. Richer profile details appear in the connection list after mutual Connect.
- Connect is a private match-scoped vote independent from Extend. The first vote receives only a private pending acknowledgment; mutual Connect persists a canonical encounter-backed connection and publishes `connection.created` once.
- Connections can be listed and removed. Blocking ends any active relationship, excludes the pair from discovery, and is manageable from the profile screen.
- Reports require a server-validated active encounter or connection. Categories and details are bounded, repeated reports for the same reporter/context are idempotent, and the database requires exactly one report context.
- Desktop navigation and four-item mobile bottom navigation link Discover, Connections and Profile. Live encounters and established connections expose report/block controls in the shared soft neobrutalist social UI.

Remote verification passed: the complete Go race and vet suite; migration replay; real PostgreSQL profile, encounter, connection, report, block and unblock lifecycle; unauthenticated access rejection; exact-Origin mutation rejection; arbitrary relationship-target rejection; private/mutual Connect with exactly one durable connection; existing Redis lifecycle and cross-instance signaling tests; TURN UDP/TCP allocation; Pion normal and forced-relay media/DataChannels; and frontend clean install, typecheck, lint and production build. The rebuilt deployment reports PostgreSQL, Redis and schema readiness; all six long-running services are healthy; and local gateway plus public Cloudflare health checks pass.

Still requires a manual two-account browser acceptance pass for private Connect visibility, mutual connection creation, richer profile display, remove/block/unblock, and both live and established-connection report dialogs. Moderation review tools, automated reputation effects, bans and abuse-rate controls remain Milestone 7 work.

## Account-first UX update (2026-09-06)

The discovery preferences now belong to the durable profile. A new account is guided to You until display name, one language and a conversation intent are saved; Discover then uses those saved values and presents one clear “Meet someone” action. Interests remain optional profile context and are sent to discovery from the saved profile.

The product shell now uses Discover, Messages, Posts and You as the four primary destinations. Mobile uses icon-only bottom navigation with accessible labels and an active marker; tablet uses a compact icon rail; desktop uses a labeled navigation rail. You includes the profile overview, connection and inbox shortcuts, preference editing, blocked people, and a settings drawer with logout. Messages and Posts are intentionally labeled as coming soon until persistent communication and publishing are implemented.

Remote frontend typecheck, lint and production build passed after the navigation/profile migration. Backend migration replay, race tests, vet, social lifecycle and discovery integration passed. The deployed public shell and health endpoints pass; an authenticated visual review of the new profile and navigation remains a manual browser check because no signed-in browser account was available to the verifier.

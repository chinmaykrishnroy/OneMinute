# Communication repair and navigation verification — 2026-09-07

All builds and automated tests ran in containers on `oneminute`. No app stack or test workloads ran on the local workstation.

## Executed

- Frontend clean install, UTF-8 guard, TypeScript, ESLint and production build.
- Full Go race suite and vet; real PostgreSQL/Redis integration including auth, discovery, migrations, signaling, social graph and communication; TURN UDP/TCP; Pion direct and forced-relay audio/DataChannels.
- Moments integration: authentication, empty-body rejection, three-active-item quota, author/current-connection visibility, outsider exclusion and deletion denial, publication-time audience, query-time expiry, owner deletion and blocked access denial.
- Chromium production-build fixture tests at 393×852, 820×1180 and 1440×1000: Discover, Messages, Moments, You and Settings title/header consistency, no horizontal overflow, back navigation, system dark appearance. Screenshots stored under remote `artifacts/browser` and reviewed for phone inbox/profile and desktop dark Moments.
- Real CallOverlay with two synthetic identities and fake cameras/microphones: caller and callee both decode video, receiver camera off/on resumes at the caller, and an audio call upgrades to two-way video. Production components and WebRTC run; test API/WebSocket fixtures replace application signaling. Separate Go integration validates real authorized signaling.
- Camera pipeline tests: timestamp-preserving processor and fresh-frame canvas fallback both exchange video and resume after camera off/on. An intentional 700 ms main-thread stall does not leave an accumulating video backlog. The final Playwright 1.63 run measured 78 ms on the preferred path and 31.5 ms on the fallback after the stall. These are test-host observations, not physical-device latency guarantees.
- Final appearance checks assert `#031009` as the system-dark page background, hidden scrollbar styling, and working wheel and PageDown scrolling on the profile page. Touch/trackpad hardware gestures remain part of physical acceptance.

## Remaining physical acceptance

Deployment: runtime revision `06df5d5` was pulled from `origin/main`, rebuilt and deployed on `oneminute`. Gateway, PostgreSQL, Redis, server, TURN and web report healthy. Readiness confirms PostgreSQL, Redis and schema; public Cloudflare `/healthz` passes, and unsigned `/v1/moments` returns 401. The remote working tree is clean. Environment files and durable volumes were preserved.

Two real signed-in accounts on representative phone/desktop hardware must verify camera permissions, microphone/video synchronization under sustained load, front/rear switching, autoplay behavior, orientation, and calls across separate networks. Safari/Firefox compatibility, mobile backgrounding and bandwidth-dependent resolution changes need device acceptance. Closed-app Web Push/native push and private media attachments remain future milestone work.

Moments is text only. The server excludes expired rows immediately and periodically deletes them; database backups have separate retention. Browser fixtures do not claim real push delivery or real Google-account end-to-end coverage.

# Test OneMinute through Cloudflare

The app runs on **llm-04**, in `/home/roy/OneMinute`. Its existing cloudflared service was active when checked on 2026-09-05. Reuse that connector. The current application is a development networking lab, before Google login and random matchmaking.

## Test now through SSH

Run this on your workstation and leave the terminal open:

```sh
ssh -N -o ExitOnForwardFailure=yes -L 127.0.0.1:3000:127.0.0.1:3000 -L 127.0.0.1:8080:127.0.0.1:8080 llm-04
```

Open http://localhost:3000/lab in two browser tabs. If a forward is already running, use it instead of starting a second process on the same ports.

1. In tab A, click **Create room**, choose **Test pattern (no camera)** and click **Join room**.
2. Copy the room code to tab B, choose the same media source and join.
3. Both should show **connected**, moving remote video, increasing received-media bytes and **DataChannel open**.
4. Send a message each way. Toggle mute/camera, restore them, and click Leave. Both peers should end the call and clear video/chat. Use a new room to reconnect.
5. To test hardware, repeat with **Camera and microphone**, grant permission and use headphones. Two tabs can share a camera on some systems; two devices provide a better hardware test.

The services run remotely; the browsers capture/play media on the devices using the app. A same-device direct connection verifies browser negotiation but does not prove internet NAT traversal. SSH forwards HTTP/WebSocket traffic here, not TURN UDP.

## Cloudflare dashboard routes

Replace `oneminute.example.com` with a hostname on your Cloudflare domain. Before exposing the development lab, create a Cloudflare Access self-hosted application for that entire hostname and allow only your tester identities. The lab itself has no account authentication. This Access gate is temporary testing access, not the planned Go application sessions.

In **Networking → Tunnels**, select the existing tunnel whose connector runs on llm-04. Under **Routes → Add route → Published application**, configure these routes in this order:

| Order | Public hostname | Path expression | Service type | Service URL |
| --- | --- | --- | --- | --- |
| 1 | `oneminute.example.com` | `^/v1/.*` | HTTP | `127.0.0.1:8080` |
| 2 | `oneminute.example.com` | Leave empty | HTTP | `127.0.0.1:3000` |

The first route serves the Go API, including `/v1/lab/ws`. The second serves Next.js, including `/lab`. Rules match from top to bottom; put the API rule before the catch-all web rule and preserve the incoming path. Both routes use the same hostname so the browser, API and Access session share an origin. WebSocket upgrades use the API route automatically; they do not need a separate TCP service. Keep WebSockets enabled for the domain, and do not add a cache rule for `/v1/*`.

The connector runs on the VM host, so these service addresses are the VM's loopback ports. Do not route to your workstation, the Docker service name `web`, PostgreSQL, Redis or TURN health port 8081.

Edit `/home/roy/OneMinute/.env` on llm-04, preserving its secrets:

```dotenv
APP_ENV=development
RTC_LAB_ENABLED=true
WEB_ORIGIN=https://oneminute.example.com
API_PUBLIC_URL=https://oneminute.example.com
```

Apply runtime configuration:

```sh
cd /home/roy/OneMinute
sudo -n docker compose -f compose.yaml -f compose.remote.yaml up -d --force-recreate --wait server web
```

Open `https://oneminute.example.com/lab`, pass the Access gate, and repeat the two-peer test. On two devices, enter the same fresh room code. Production mode intentionally disables this lab. These instructions do not configure a public production release.

When switching back to SSH-only browser testing, restore `WEB_ORIGIN=http://localhost:3000` and `API_PUBLIC_URL=http://localhost:8080`, then recreate server/web again. The API enforces the configured exact origin.

## TURN is a separate network path

A published HTTP Cloudflare Tunnel route carries the page and signaling. It does **not** expose the public UDP TURN relay needed by ordinary WebRTC browsers. Publishing a TCP application would require client-side cloudflared and does not solve this requirement.

For public relay tests:

1. Assign/reserve a reachable public IPv4 for llm-04. The VM metadata reported **34.100.213.131** on 2026-09-05; verify it before using it, since it may be ephemeral. Its current TURN configuration advertises private `10.160.3.131`, which public browsers cannot reach.
2. Set `TURN_PUBLIC_IP` in the remote `.env` to that public IPv4. This implementation uses the numeric IP for both the browser's TURN URL and relay candidates.
3. Allow inbound **UDP 3478**, **TCP 3478**, and **UDP 49160–49180** through the cloud VPC firewall and any host firewall, targeting this VM. Allow return traffic. Use the actual configured port range if changed.
4. Recreate the services that consume the TURN address:

```sh
sudo -n docker compose -f compose.yaml -f compose.remote.yaml up -d --force-recreate --wait server turn
```

An optional `turn.example.com` DNS record must be **A → VM public IPv4, DNS only (gray cloud)**. It is not a tunnel route. The current code continues to advertise the numeric IP; adding the DNS record alone does not change that behavior.

Create a new room, select **Force TURN relay** on both peers before joining, and verify `relay ↔ relay`, moving video, increasing received bytes and bidirectional chat. Repeat with devices on different networks, such as home Wi-Fi and cellular. TURN credentials are issued temporarily by the API; never put the shared TURN secret in the dashboard or browser. The current TCP fallback is TURN over TCP on 3478, not TURN over TLS on 443.

## Troubleshooting

- **502:** check the selected connector is llm-04 and the origin services are healthy. On llm-04, run `curl -f http://127.0.0.1:3000/healthz` and `curl -f http://127.0.0.1:8080/readyz`.
- **Create room fails or signaling is rejected:** check `/v1/*` routes to 8080 before the catch-all, both origin settings use the exact HTTPS hostname, and the browser passed Access.
- **Page/chat signaling works but video will not connect:** check TURN's advertised address and direct firewall ports. HTTP tunnel health does not prove media connectivity.
- **Lab returns 404:** check `RTC_LAB_ENABLED=true`, `APP_ENV=development`, and recreate server/web after editing `.env`.
- **Old room will not reconnect:** rooms expire after ten minutes and end when a participant leaves. Create a fresh room.

## References

- [Cloudflare dashboard tunnel setup](https://developers.cloudflare.com/tunnel/setup/)
- [Ingress path matching and rule order](https://developers.cloudflare.com/cloudflare-one/networks/connectors/cloudflare-tunnel/do-more-with-tunnels/local-management/configuration-file/)
- [WebSocket support](https://developers.cloudflare.com/network/websockets/)
- [Published application protocols](https://developers.cloudflare.com/cloudflare-one/networks/connectors/cloudflare-tunnel/routing-to-tunnel/protocols/)
- [DNS proxy status](https://developers.cloudflare.com/dns/proxy-status/)
- [Protect a self-hosted application with Access](https://developers.cloudflare.com/cloudflare-one/access-controls/applications/http-apps/self-hosted-public-app/)

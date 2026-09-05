# Cloudflare deployment and browser testing

## Current deployment

- Machine: `ssh oneminute`, project `/home/roy/OneMinute`.
- Public lab: https://oneminute.prefect-sys.online/lab.
- Cloudflare published application: **oneminute.prefect-sys.online**, path empty, service **HTTP**, URL **127.0.0.1:3000**.
- Keep the existing tunnel/connector on the new machine. No second hostname or path route is needed with `compose.public.yaml`.

The gateway on port 3000 sends `/v1/*` to Go on container port 8080 and everything else to Next.js on container port 3000. WebSocket signaling uses the same HTTPS origin. The gateway preserves paths and supports upgrades. Do not configure a separate TCP tunnel for WebSockets.

The remote `.env` contains:

```dotenv
APP_ENV=development
RTC_LAB_ENABLED=true
WEB_ORIGIN=https://oneminute.prefect-sys.online
API_PUBLIC_URL=https://oneminute.prefect-sys.online
TURN_PUBLIC_IP=35.234.222.18
```

Preserve the generated database, Redis and TURN secrets. Deploy with all three files:

```sh
cd /home/roy/OneMinute
sudo -n docker compose -f compose.yaml -f compose.remote.yaml -f compose.public.yaml up --build -d --wait
```

Cloudflare Tunnel provides HTTPS for the browser. The current milestone is a development lab with temporary random room codes, before application login. Anyone holding a room code can occupy an available slot; rooms expire after ten minutes. Cloudflare Access can restrict the entire hostname to tester identities while application identity is being built. Keep caching disabled for `/v1/*` and leave WebSockets enabled for the domain.

## Test a call

1. Open the public lab in two tabs or devices. In the first, click **Create room**, choose **Test pattern (no camera)** and click **Join room**.
2. Enter that code in the second tab, select the same source and join.
3. Verify **connected**, moving remote video, increasing received-media bytes and **DataChannel open**. Send a message in each direction.
4. Toggle mute/camera, restore them, and leave. Both peers should end and clear video/chat. Create a fresh room to reconnect.
5. For relay verification, select **Force TURN relay** on both peers before joining. The selected pair must show `relay ↔ relay`. Repeat the video/chat checks.
6. For hardware acceptance, choose **Camera and microphone**, grant permissions and use headphones. Two devices on separate networks, such as Wi-Fi and cellular, provide the best test of microphones, cameras and NAT behavior.

The automated browser checks use generated video and silent audio. They verify real WebRTC transport without recording a person. Physical camera/microphone capture and audible playback still need a manual device check.

## TURN ports

The dedicated VM's public IPv4 was **35.234.222.18** when checked on 2026-09-05. The owner opened 3478 and 49160–49180. Forced browser relay succeeded with this address. Maintain:

| Purpose | Direct inbound traffic to VM |
| --- | --- |
| STUN/TURN | UDP 3478 and TCP 3478 |
| Allocated relay ports | UDP 49160–49180 |

Reserve/recheck the public IP before relying on it long term. A published HTTP tunnel cannot carry ordinary browser TURN UDP. An optional `turn.prefect-sys.online` A record should be DNS only (gray cloud), pointing to the VM public IP; the implementation currently advertises the numeric IP, so DNS is optional. TCP fallback is TURN on 3478, not TURN over TLS on 443.

Do not tunnel PostgreSQL, Redis or port 8081. Those are not browser services. The permanent TURN secret stays on the server; browsers receive expiring credentials.

## Troubleshooting

- **502:** verify the tunnel connector runs on `oneminute` and its service URL is HTTP `127.0.0.1:3000`. Check `docker compose` with all three files reports a healthy gateway, web and server.
- **Create room fails:** check the two exact HTTPS origin settings and that `compose.public.yaml` is included. Tunneling Next.js alone does not route the Go API.
- **Page works but media fails:** check the advertised TURN IP and direct firewall ports. A healthy HTTP tunnel does not prove media connectivity.
- **Lab 404:** check development mode and RTC_LAB_ENABLED=true, then recreate web/server.
- **Cannot rejoin:** use a fresh room after leave or ten-minute expiry.

Deployment/test commands: [remote development](remote-development.md).

## References

- [Cloudflare published application setup](https://developers.cloudflare.com/tunnel/setup/)
- [Cloudflare WebSocket support](https://developers.cloudflare.com/network/websockets/)
- [Published application protocols](https://developers.cloudflare.com/cloudflare-one/networks/connectors/cloudflare-tunnel/routing-to-tunnel/protocols/)
- [DNS proxy status](https://developers.cloudflare.com/dns/proxy-status/)
- [Cloudflare Access](https://developers.cloudflare.com/cloudflare-one/access-controls/applications/http-apps/self-hosted-public-app/)
- [Caddy reverse proxy and WebSocket support](https://caddyserver.com/docs/caddyfile/directives/reverse_proxy)

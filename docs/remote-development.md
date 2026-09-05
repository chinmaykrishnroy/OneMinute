# Dedicated development and deployment host

Run application builds, containers and tests on **ssh oneminute**, in `/home/roy/OneMinute`. The public networking lab is https://oneminute.prefect-sys.online/lab. Local source editing, Git and browser use are allowed; application compute stays on the dedicated host.

Docker requires `sudo -n` for roy. The repository remote is https://github.com/chinmaykrishnroy/OneMinute.git, branch main. Fresh deployment secrets live only in the remote `.env` (mode 0600).

## Update and start

```sh
ssh oneminute
cd /home/roy/OneMinute
git pull --ff-only origin main
sudo -n docker compose -f compose.yaml -f compose.remote.yaml -f compose.public.yaml up --build -d --wait --wait-timeout 240
```

The three Compose files are cumulative: base dependencies and services, Linux host-network TURN, then the single-port HTTP gateway. Preserve `.env` and database volumes on updates. Do not run initialization over an existing environment or copy secrets into Git.

## Verify

```sh
sudo -n docker compose -f compose.yaml -f compose.remote.yaml -f compose.public.yaml -f compose.test.yaml --profile test run --rm verify-go
sudo -n docker compose -f compose.yaml -f compose.remote.yaml -f compose.public.yaml -f compose.test.yaml --profile test run --rm verify-web
curl -f https://oneminute.prefect-sys.online/healthz
```

The Go service runs race detection, vet, real PostgreSQL/Redis integration, distributed signaling checks, STUN/TURN allocation and Pion media/DataChannel tests. TEST_WEB_ORIGIN follows WEB_ORIGIN so the same checks work with the public deployment's exact origin. Test toolchains live in containers.

## Network topology

The existing cloudflared host service sends `oneminute.prefect-sys.online` to `http://127.0.0.1:3000`. Caddy receives that one port and forwards `/v1/*` unchanged to Go at `server:8080`, including WebSocket upgrades. Other paths go to `web:3000`. Next.js has no directly published host port in this deployment.

TURN advertises **35.234.222.18**, the new VM's public IPv4 observed on 2026-09-05. Reserve or recheck that address if the VM is stopped or recreated. TURN uses direct UDP/TCP 3478 and UDP relay ports 49160–49180. Cloudflare carries web/signaling, not TURN media. The TURN health listener, API, PostgreSQL and Redis bind only to host loopback where published.

See [Cloudflare testing](cloudflare-testing.md) for the current settings and manual call checks. Google login and application sessions start in Milestone 2; the current lab uses temporary room capabilities.

## Previous host

llm-04 is retired from the OneMinute workflow after verification on the dedicated host. Its application resources are removed only after confirming the new deployment works. Cleanup is scoped to the OneMinute workspace, transfer artifacts and Docker resources; unrelated cloudflared/Tailscale/LLM services and files must remain intact. See [verification](verification.md) for the actual cleanup outcome.

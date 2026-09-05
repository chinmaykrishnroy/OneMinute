# Remote development: llm-04

Per the project owner's instruction, run application builds and tests on `ssh llm-04`, not on the local workstation. The workspace is `/home/roy/OneMinute`. Docker requires `sudo -n` for the roy account. No global Go or Node install is required.

The previous local Compose stack has been stopped and removed; its durable development volume is preserved. No previous OneMinute deployment was found on llm-04 in /home, /root, /opt, /srv or /ephemeral, and Docker had no application containers or volumes to clean. Do not remove unrelated services or model files.

On the remote host:

```sh
cd /home/roy/OneMinute
sh scripts/remote-init.sh
sudo -n docker compose -f compose.yaml -f compose.remote.yaml up --build --detach --wait --wait-timeout 180
sudo -n docker compose -f compose.yaml -f compose.remote.yaml -f compose.test.yaml --profile test run --rm verify-go
sudo -n docker compose -f compose.yaml -f compose.remote.yaml -f compose.test.yaml --profile test run --rm verify-web
```

The remote .env uses fresh secrets and a reachable host interface address for TURN. It is not copied from the workstation or committed. The API/web remain loopback-bound. HTTP health checks can run over SSH using curl; browsers need an SSH forward or an intentionally configured HTTPS endpoint. Google login and public deployment are later work.

To test in a browser, forward web/API ports from the remote host:
```sh
ssh -L 3000:127.0.0.1:3000 -L 8080:127.0.0.1:8080 llm-04
```
The browser UI then uses localhost while application compute runs on llm-04. TURN still needs a directly reachable host/public address; a TCP SSH forward does not forward TURN UDP. For public NAT tests use the VM public IPv4 and explicit firewall rules. Do not expose the unauthenticated development lab publicly.

See [Cloudflare testing](cloudflare-testing.md) for the two-peer test procedure, exact dashboard routes, origin configuration and separate TURN firewall requirements.

Use a source archive excluding .env, .git, node_modules, .next and artifacts when copying an uncommitted test slice. After verification, commit and push a coherent checkpoint to origin/main without attribution trailers.

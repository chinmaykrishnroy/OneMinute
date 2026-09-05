# ICE and TURN
Media topology: browser ↔ browser over DTLS/SRTP; TURN is the fallback packet relay. Go application servers exchange signaling only. Text conversation uses RTCDataChannel and is not stored.

Pion TURN runs independently on UDP and TCP 3478. TCP is the browser-to-TURN transport fallback while relay media remains UDP. TURN/TLS for restrictive networks and production certificates are a later hardening step; TCP 3478 is not TLS 443.

Set TURN_PUBLIC_IP to the address reachable by the testing browsers. 127.0.0.1 works only on the same host. For LAN tests use the host LAN IPv4; for the VM use its public IPv4 and DNS-only TURN hostname. HTTP camera permission works on localhost; remote browser tests need HTTPS.

Publish the entire configured UDP relay range with identical external/internal ports. Open UDP/TCP 3478 and that UDP range in the firewall. Do not put TURN behind ordinary Cloudflare Tunnel. Docker Desktop NAT and upstream NAT can affect candidate reachability; a healthy HTTP endpoint does not prove relay connectivity.

Temporary credentials use the TURN REST convention: username `expiryUnix:subject`, password base64(HMAC-SHA1(secret, username)). Pion verifies the derived long-term auth key. Credentials last at most one hour; the browser never sees TURN_SECRET. The authenticated backend will issue short-lived ICE configuration through the provider boundary. Expired credentials prevent new allocations/refreshes; existing allocations are not forcibly closed at credential expiry.

Private/loopback peer permissions are allowed only by explicit local configuration. Production rejects that option and nonpublic relay addresses. Apply host/network egress ACLs as defense in depth. Per-account allocation/bandwidth quotas, TURN TLS, secret rotation and abuse alerts remain release-hardening work.

Health: :8081/healthz indicates initialized listeners; :8081/metrics exposes auth callback counters (not successfully authenticated packets or bandwidth). Neither endpoint is exposed publicly by Compose.

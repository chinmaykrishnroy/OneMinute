#!/usr/bin/env sh
set -eu
cd "$(dirname "$0")/.."
# Docker performs toolchain work; only Python's standard library is used for secrets.
python3 - <<'PY'
from pathlib import Path
import secrets
import re
import socket
target = Path(".env")
if target.exists():
    print(".env exists; preserved")
else:
    content = Path(".env.example").read_text()
    for name in ("POSTGRES_PASSWORD", "REDIS_PASSWORD", "TURN_SECRET"):
        content = re.sub(r"^" + name + r"=.*$", name + "=" + secrets.token_hex(32), content, flags=re.M)
    content = content.replace("RTC_LAB_ENABLED=false", "RTC_LAB_ENABLED=true")
    # A route lookup obtains the host interface address without sending application data.
    with socket.socket(socket.AF_INET, socket.SOCK_DGRAM) as sock:
        sock.connect(("192.0.2.1", 9))
        host_ip = sock.getsockname()[0]
    content = content.replace("TURN_PUBLIC_IP=127.0.0.1", "TURN_PUBLIC_IP=" + host_ip)
    with target.open("x") as f:
        f.write(content)
    target.chmod(0o600)
    print("Created remote-only development configuration")
PY

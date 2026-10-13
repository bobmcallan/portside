#!/usr/bin/env bash
# Local docker deploy/update for portside.
# Builds the image from the tree, brings the compose stack up on 127.0.0.1:8888,
# and probes it. Idempotent: safe to re-run (converges via compose up -d --build).
# Never removes volumes; never binds off localhost.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

PROBE_URL="http://127.0.0.1:8888/"
MAX_ATTEMPTS=30
SLEEP_SECS=1

if ! command -v docker >/dev/null 2>&1; then
  echo "deploy: docker not found on PATH" >&2
  exit 1
fi
if ! docker compose version >/dev/null 2>&1; then
  echo "deploy: docker compose (v2) is required" >&2
  exit 1
fi

if command -v curl >/dev/null 2>&1; then
  fetch_page() { curl -fsS --max-time 2 "$PROBE_URL"; }
elif command -v wget >/dev/null 2>&1; then
  fetch_page() { wget -q -O - --timeout=2 "$PROBE_URL"; }
else
  echo "deploy: need curl or wget to probe $PROBE_URL" >&2
  exit 1
fi

# Host-side Docker socket to mount (compose.yaml reads DOCKER_SOCK). An explicit
# DOCKER_SOCK wins; else a unix:// DOCKER_HOST (e.g. rootless Docker) supplies
# it; else compose falls back to /var/run/docker.sock.
if [ -z "${DOCKER_SOCK:-}" ] && [[ "${DOCKER_HOST:-}" == unix://* ]]; then
  export DOCKER_SOCK="${DOCKER_HOST#unix://}"
fi
SOCK="${DOCKER_SOCK:-/var/run/docker.sock}"
echo "deploy: docker socket $SOCK"
# A missing source would be bind-mounted as an empty directory and the page
# would render a docker connection error, so refuse before touching the stack.
if [ ! -S "$SOCK" ]; then
  echo "deploy: $SOCK is not a socket — set DOCKER_SOCK or DOCKER_HOST to the daemon's socket" >&2
  exit 1
fi

echo "deploy: building and bringing up the stack (docker compose up -d --build)…"
# Sole mutating docker command: converging update. No down, no -v, no prune.
docker compose up -d --build

echo "deploy: probing $PROBE_URL …"
attempt=1
while [ "$attempt" -le "$MAX_ATTEMPTS" ]; do
  # Guarded so set -e does not abort on expected connection failures during boot.
  if body="$(fetch_page)"; then
    # The page renders (HTTP 200) even when the backend cannot reach the daemon,
    # so judge its content: no docker error, and at least one container row
    # (portside itself is running, so a working backend always lists one).
    if grep -q 'docker error' <<<"$body"; then
      echo "deploy: page reports a docker error — the socket mount is broken:" >&2
      grep -o 'docker error:[^<]*' <<<"$body" >&2 || true
      docker compose logs --tail=50 portside >&2 || true
      exit 1
    fi
    if ! grep -q 'data-ctr-action=' <<<"$body"; then
      echo "deploy: page lists no containers — the backend is not reading the daemon" >&2
      docker compose logs --tail=50 portside >&2 || true
      exit 1
    fi
    echo "deploy: ok — $PROBE_URL"
    exit 0
  fi
  sleep "$SLEEP_SECS"
  attempt=$((attempt + 1))
done

echo "deploy: probe failed after ${MAX_ATTEMPTS}s — last logs:" >&2
docker compose logs --tail=50 portside >&2 || true
exit 1

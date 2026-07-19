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
  probe_once() { curl -fsS -o /dev/null --max-time 2 "$PROBE_URL"; }
elif command -v wget >/dev/null 2>&1; then
  probe_once() { wget -q -O /dev/null --timeout=2 "$PROBE_URL"; }
else
  echo "deploy: need curl or wget to probe $PROBE_URL" >&2
  exit 1
fi

echo "deploy: building and bringing up the stack (docker compose up -d --build)…"
# Sole mutating docker command: converging update. No down, no -v, no prune.
docker compose up -d --build

echo "deploy: probing $PROBE_URL …"
attempt=1
while [ "$attempt" -le "$MAX_ATTEMPTS" ]; do
  # Guarded so set -e does not abort on expected connection failures during boot.
  if probe_once; then
    echo "deploy: ok — $PROBE_URL"
    exit 0
  fi
  sleep "$SLEEP_SECS"
  attempt=$((attempt + 1))
done

echo "deploy: probe failed after ${MAX_ATTEMPTS}s — last logs:" >&2
docker compose logs --tail=50 portside >&2 || true
exit 1

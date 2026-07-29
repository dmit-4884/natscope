#!/bin/sh
set -eu

# Baseline measured 2026-07-12 on the built frontend: gzipped total of
# internal/transports/grpc/dist/assets/*.js was 524923 bytes (~512.6 KiB).
# Threshold below is 15% above that baseline.
BASELINE_BYTES=524923
THRESHOLD_BYTES=603661

SCRIPT_DIR=$(cd "$(dirname "$0")" && pwd)
REPO_ROOT=$(cd "$SCRIPT_DIR/.." && pwd)
ASSETS_DIR="$REPO_ROOT/internal/transports/grpc/dist/assets"

if [ ! -d "$ASSETS_DIR" ]; then
  echo "error: $ASSETS_DIR not found — run the frontend build first" >&2
  exit 1
fi

total=0
for f in "$ASSETS_DIR"/*.js; do
  [ -f "$f" ] || continue
  sz=$(gzip -c "$f" | wc -c)
  total=$((total + sz))
done

if [ "$total" -gt "$THRESHOLD_BYTES" ]; then
  echo "error: gzipped JS bundle size ${total} bytes exceeds budget ${THRESHOLD_BYTES} bytes (baseline ${BASELINE_BYTES} bytes + 15%)" >&2
  exit 1
fi

echo "bundle size OK: ${total} gzipped bytes (budget ${THRESHOLD_BYTES} bytes, baseline ${BASELINE_BYTES} bytes)"

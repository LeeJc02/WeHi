#!/usr/bin/env bash
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/local-env.sh"

export NO_PROXY="127.0.0.1,localhost"
export no_proxy="127.0.0.1,localhost"

curl --noproxy '*' -fsS "http://127.0.0.1:${API_PORT}/api/v1/system/ready" >/dev/null
(cd "$ROOT_DIR/frontend" && npm run smoke -- "http://127.0.0.1:${AUTH_PORT}" "http://127.0.0.1:${API_PORT}")
node "$ROOT_DIR/scripts/runtime_smoke.mjs" "http://127.0.0.1:${AUTH_PORT}" "http://127.0.0.1:${API_PORT}" "ws://127.0.0.1:${REALTIME_PORT}/ws"

echo "Local smoke passed"

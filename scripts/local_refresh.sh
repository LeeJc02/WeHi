#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
source "$ROOT_DIR/scripts/local-env.sh"

"$ROOT_DIR/scripts/local_up.sh"

for service in auth-service api-service realtime-service frontend; do
  "$ROOT_DIR/scripts/local_restart.sh" "$service"
done

cat <<EOF
Local workspace refreshed.
Auth:      http://127.0.0.1:${AUTH_PORT:-19081}
API:       http://127.0.0.1:${API_PORT:-19082}
Realtime:  ws://127.0.0.1:${REALTIME_PORT:-19083}/ws
Frontend:  http://127.0.0.1:${FRONTEND_PORT:-5173}
EOF

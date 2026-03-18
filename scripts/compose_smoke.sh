#!/usr/bin/env bash
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/compose_env.sh"
load_state

compose_cmd exec -T frontend npm run smoke -- http://auth-service:8081 http://api-service:8082
compose_cmd exec -T frontend node /app/scripts/runtime_smoke.mjs \
  http://auth-service:8081 \
  http://api-service:8082 \
  ws://realtime-service:8083/ws

echo "Compose smoke passed"

#!/usr/bin/env bash
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/compose_env.sh"
load_state

echo "App ports: auth=${AUTH_PORT} api=${API_PORT} realtime=${REALTIME_PORT} frontend=${FRONTEND_PORT}"

COMPOSE_PARALLEL_LIMIT=1 compose_cmd up -d --build --wait --remove-orphans

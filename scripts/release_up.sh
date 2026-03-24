#!/usr/bin/env bash
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/compose_env.sh"
load_state

: "${IMAGE_PREFIX:?set IMAGE_PREFIX, for example ghcr.io/<owner>/<repo>}"

echo "Release image prefix: ${IMAGE_PREFIX}"
echo "Release image tag: ${IMAGE_TAG:-latest}"
echo "App ports: auth=${AUTH_PORT} api=${API_PORT} realtime=${REALTIME_PORT} frontend=${FRONTEND_PORT}"

compose_release_cmd pull
compose_release_cmd up -d --wait --remove-orphans

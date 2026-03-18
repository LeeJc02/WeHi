#!/usr/bin/env bash
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/compose_env.sh"

if [[ -f "$COMPOSE_STATE_FILE" ]]; then
  load_state
fi

compose_cmd down --remove-orphans || true
rm -f "$COMPOSE_STATE_FILE"

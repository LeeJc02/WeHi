#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
RUNTIME_DIR="${RUNTIME_DIR:-$ROOT_DIR/.runtime}"
COMPOSE_STATE_FILE="${COMPOSE_STATE_FILE:-$RUNTIME_DIR/compose-mode.env}"

port_in_use() {
  local port="$1"
  lsof -nP -iTCP:"$port" -sTCP:LISTEN >/dev/null 2>&1
}

write_state() {
  mkdir -p "$RUNTIME_DIR"
  cat >"$COMPOSE_STATE_FILE" <<EOF
AUTH_PORT=${AUTH_PORT}
API_PORT=${API_PORT}
REALTIME_PORT=${REALTIME_PORT}
FRONTEND_PORT=${FRONTEND_PORT}
EOF
}

select_ports() {
  local candidates=(
    "28081 28082 28083 25173"
    "38081 38082 38083 35173"
    "48081 48082 48083 45173"
  )
  local entry auth api realtime frontend

  for entry in "${candidates[@]}"; do
    read -r auth api realtime frontend <<<"$entry"
    if ! port_in_use "$auth" && ! port_in_use "$api" && ! port_in_use "$realtime" && ! port_in_use "$frontend"; then
      AUTH_PORT="$auth"
      API_PORT="$api"
      REALTIME_PORT="$realtime"
      FRONTEND_PORT="$frontend"
      write_state
      export AUTH_PORT API_PORT REALTIME_PORT FRONTEND_PORT
      return 0
    fi
  done

  echo "No free Docker app port set is available." >&2
  exit 1
}

load_state() {
  if [[ ! -f "$COMPOSE_STATE_FILE" ]]; then
    select_ports
    return
  fi

  # shellcheck disable=SC1090
  source "$COMPOSE_STATE_FILE"
  export AUTH_PORT API_PORT REALTIME_PORT FRONTEND_PORT
}

export ROOT_DIR RUNTIME_DIR COMPOSE_STATE_FILE

compose_cmd() {
  docker compose -f deploy/compose/docker-compose.yml "$@"
}

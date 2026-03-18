#!/usr/bin/env bash
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/local-env.sh"

MYSQL_SOCKET_PROTOCOL="socket"

is_port_listening() {
  local port="$1"
  lsof -iTCP:"$port" -sTCP:LISTEN >/dev/null 2>&1
}

listener_pids() {
  local port="$1"
  lsof -tiTCP:"$port" -sTCP:LISTEN 2>/dev/null || true
}

stop_port_listener() {
  local port="$1"
  local name="$2"
  local pids
  pids="$(listener_pids "$port")"
  if [ -z "$pids" ]; then
    return
  fi

  kill $pids >/dev/null 2>&1 || true
  sleep 1

  if is_port_listening "$port"; then
    pids="$(listener_pids "$port")"
    if [ -n "$pids" ]; then
      kill -9 $pids >/dev/null 2>&1 || true
    fi
  fi

  echo "Stopped ${name} port listener"
}

shutdown_pid() {
  local pid_file="$1"
  local name="$2"
  if [ ! -f "$pid_file" ]; then
    return
  fi
  local pid
  pid="$(cat "$pid_file")"
  if kill -0 "$pid" >/dev/null 2>&1; then
    kill "$pid" >/dev/null 2>&1 || true
    wait "$pid" 2>/dev/null || true
    echo "Stopped ${name}"
  fi
  rm -f "$pid_file"
}

shutdown_pid "$PID_DIR/frontend.pid" "frontend"
shutdown_pid "$PID_DIR/realtime-service.pid" "realtime-service"
shutdown_pid "$PID_DIR/api-service.pid" "api-service"
shutdown_pid "$PID_DIR/auth-service.pid" "auth-service"

stop_port_listener "$FRONTEND_PORT" "frontend"
stop_port_listener "$REALTIME_PORT" "realtime-service"
stop_port_listener "$API_PORT" "api-service"
stop_port_listener "$AUTH_PORT" "auth-service"

if [ -f "$REDIS_PID_FILE" ]; then
  pid="$(cat "$REDIS_PID_FILE")"
  if kill -0 "$pid" >/dev/null 2>&1; then
    kill "$pid" >/dev/null 2>&1 || true
    wait "$pid" 2>/dev/null || true
    echo "Stopped local Redis"
  fi
  rm -f "$REDIS_PID_FILE"
fi

if [ -f "$MYSQL_PID_FILE" ]; then
  MYSQLADMIN_BIN="${MYSQLADMIN_BIN:-$(command -v mysqladmin)}"
  if [ -n "$MYSQLADMIN_BIN" ] && "$MYSQLADMIN_BIN" --protocol="$MYSQL_SOCKET_PROTOCOL" --socket="$MYSQL_SOCKET" -uroot ping >/dev/null 2>&1; then
    "$MYSQLADMIN_BIN" --protocol="$MYSQL_SOCKET_PROTOCOL" --socket="$MYSQL_SOCKET" -uroot shutdown >/dev/null 2>&1 || true
    echo "Stopped local MySQL"
  fi
  rm -f "$MYSQL_PID_FILE"
fi

DOCKER_BIN="${DOCKER_BIN:-$(command -v docker || true)}"
if [ -f "$MYSQL_DOCKER_STATE_FILE" ] && [ -n "$DOCKER_BIN" ]; then
  if "$DOCKER_BIN" container inspect "$MYSQL_DOCKER_CONTAINER" >/dev/null 2>&1; then
    "$DOCKER_BIN" rm -f "$MYSQL_DOCKER_CONTAINER" >/dev/null 2>&1 || true
    echo "Stopped Docker MySQL"
  fi
  rm -f "$MYSQL_DOCKER_STATE_FILE"
fi

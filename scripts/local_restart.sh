#!/usr/bin/env bash
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/local-env.sh"

BACKEND_DIR="$ROOT_DIR/backend"

SERVICE_NAME="${1:-}"

if [ -z "$SERVICE_NAME" ]; then
  echo "Usage: $0 <auth-service|api-service|realtime-service|frontend>" >&2
  exit 1
fi

is_port_listening() {
  local port="$1"
  lsof -iTCP:"$port" -sTCP:LISTEN >/dev/null 2>&1
}

listener_pids() {
  local port="$1"
  lsof -tiTCP:"$port" -sTCP:LISTEN 2>/dev/null || true
}

wait_for_port_release() {
  local port="$1"
  for _ in {1..30}; do
    if ! is_port_listening "$port"; then
      return 0
    fi
    sleep 1
  done
  return 1
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

  if ! wait_for_port_release "$port"; then
    pids="$(listener_pids "$port")"
    if [ -n "$pids" ]; then
      kill -9 $pids >/dev/null 2>&1 || true
    fi
  fi

  if ! wait_for_port_release "$port"; then
    echo "Failed to stop ${name} on ${port}" >&2
    exit 1
  fi
}

stop_pid() {
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

stop_service() {
  local pid_file="$1"
  local name="$2"
  local port="$3"

  stop_pid "$pid_file" "$name"
  stop_port_listener "$port" "$name"
}

start_service() {
  local name="$1"
  local path="$2"
  local port="$3"
  local pid_file="$PID_DIR/${name}.pid"
  local log_file="$LOG_DIR/${name}.log"
  local binary="$BIN_DIR/${name}"

  stop_service "$pid_file" "$name" "$port"

  (
    cd "$BACKEND_DIR"
    env \
      APP_PORT="$port" \
      MYSQL_DSN="$MYSQL_DSN" \
      REDIS_ADDR="$REDIS_ADDR" \
      RABBITMQ_URL="$RABBITMQ_URL" \
      ELASTICSEARCH_URL="$ELASTICSEARCH_URL" \
      JWT_SECRET="$JWT_SECRET" \
      CORS_ORIGINS="$CORS_ORIGINS" \
      go build -o "$binary" "$path"
  )

  nohup bash -lc "cd '$BACKEND_DIR' && APP_PORT='$port' MYSQL_DSN='$MYSQL_DSN' REDIS_ADDR='$REDIS_ADDR' RABBITMQ_URL='$RABBITMQ_URL' ELASTICSEARCH_URL='$ELASTICSEARCH_URL' JWT_SECRET='$JWT_SECRET' CORS_ORIGINS='$CORS_ORIGINS' '$binary'" >"$log_file" 2>&1 < /dev/null &

  local pid=$!
  echo "$pid" >"$pid_file"

  for _ in {1..30}; do
    if is_port_listening "$port"; then
      echo "Started ${name} on ${port}"
      return
    fi
    sleep 1
  done

  echo "Failed to start ${name}. Check ${log_file}" >&2
  exit 1
}

start_frontend() {
  local pid_file="$PID_DIR/frontend.pid"
  local log_file="$LOG_DIR/frontend.log"

  stop_service "$pid_file" "frontend" "$FRONTEND_PORT"

  if [ ! -d "$ROOT_DIR/frontend/node_modules" ]; then
    (cd "$ROOT_DIR/frontend" && npm install)
  fi

  nohup bash -lc "cd '$ROOT_DIR/frontend' && NEXT_PUBLIC_AUTH_BASE_URL='http://127.0.0.1:${AUTH_PORT}' NEXT_PUBLIC_API_BASE_URL='http://127.0.0.1:${API_PORT}' NEXT_PUBLIC_REALTIME_BASE_URL='ws://127.0.0.1:${REALTIME_PORT}' npm run dev -- --hostname 127.0.0.1 --port '$FRONTEND_PORT'" >"$log_file" 2>&1 < /dev/null &

  local pid=$!
  echo "$pid" >"$pid_file"

  for _ in {1..30}; do
    if is_port_listening "$FRONTEND_PORT"; then
      echo "Started frontend on ${FRONTEND_PORT}"
      return
    fi
    sleep 1
  done

  echo "Failed to start frontend. Check ${log_file}" >&2
  exit 1
}

case "$SERVICE_NAME" in
  auth-service)
    start_service "auth-service" "./services/auth" "$AUTH_PORT"
    ;;
  api-service)
    start_service "api-service" "./services/api" "$API_PORT"
    ;;
  realtime-service)
    start_service "realtime-service" "./services/realtime" "$REALTIME_PORT"
    ;;
  frontend)
    start_frontend
    ;;
  *)
    echo "Unsupported service: $SERVICE_NAME" >&2
    echo "Expected one of: auth-service, api-service, realtime-service, frontend" >&2
    exit 1
    ;;
esac

#!/usr/bin/env bash
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/local-env.sh"

BACKEND_DIR="$ROOT_DIR/backend"

MYSQLD_BIN="${MYSQLD_BIN:-$(command -v mysqld)}"
MYSQLD_SAFE_BIN="${MYSQLD_SAFE_BIN:-$(command -v mysqld_safe || true)}"
MYSQL_BIN="${MYSQL_BIN:-$(command -v mysql)}"
MYSQLADMIN_BIN="${MYSQLADMIN_BIN:-$(command -v mysqladmin)}"
REDIS_SERVER_BIN="${REDIS_SERVER_BIN:-$(command -v redis-server)}"
REDIS_CLI_BIN="${REDIS_CLI_BIN:-$(command -v redis-cli)}"
DOCKER_BIN="${DOCKER_BIN:-$(command -v docker || true)}"
MYSQL_SOCKET_PROTOCOL="socket"
MYSQL_INIT_FILE="$RUNTIME_DIR/mysql-init.sql"
LOCAL_MYSQL_STARTED=0

need_cmd() {
  if [ -z "${2}" ]; then
    echo "Missing required command: ${1}" >&2
    exit 1
  fi
}

need_cmd "mysql" "$MYSQL_BIN"
need_cmd "mysqladmin" "$MYSQLADMIN_BIN"
need_cmd "redis-server" "$REDIS_SERVER_BIN"
need_cmd "redis-cli" "$REDIS_CLI_BIN"

is_port_listening() {
  local port="$1"
  lsof -iTCP:"$port" -sTCP:LISTEN >/dev/null 2>&1
}

mysql_root_ping() {
  "$MYSQLADMIN_BIN" -h"$MYSQL_HOST" -P"$MYSQL_PORT" -uroot ping >/dev/null 2>&1 || \
    "$MYSQLADMIN_BIN" -h"$MYSQL_HOST" -P"$MYSQL_PORT" -uroot "-p$MYSQL_ROOT_PASSWORD" ping >/dev/null 2>&1
}

mysql_root_exec() {
  if "$MYSQL_BIN" -h"$MYSQL_HOST" -P"$MYSQL_PORT" -uroot -e "SELECT 1" >/dev/null 2>&1; then
    "$MYSQL_BIN" -h"$MYSQL_HOST" -P"$MYSQL_PORT" -uroot "$@"
    return
  fi
  "$MYSQL_BIN" -h"$MYSQL_HOST" -P"$MYSQL_PORT" -uroot "-p$MYSQL_ROOT_PASSWORD" "$@"
}

mysql_app_ping() {
  "$MYSQL_BIN" -h"$MYSQL_HOST" -P"$MYSQL_PORT" -u"$MYSQL_USER" -p"$MYSQL_PASSWORD" -e "SELECT 1" "$MYSQL_DATABASE" >/dev/null 2>&1
}

docker_available() {
  [ -n "$DOCKER_BIN" ] && "$DOCKER_BIN" info >/dev/null 2>&1
}

docker_mysql_image_available() {
  docker_available && "$DOCKER_BIN" image inspect "$MYSQL_DOCKER_IMAGE" >/dev/null 2>&1
}

docker_mysql_container_exists() {
  [ -n "$DOCKER_BIN" ] && "$DOCKER_BIN" container inspect "$MYSQL_DOCKER_CONTAINER" >/dev/null 2>&1
}

clear_mysql_docker_state() {
  rm -f "$MYSQL_DOCKER_STATE_FILE"
}

mark_mysql_docker_state() {
  printf 'docker\n' >"$MYSQL_DOCKER_STATE_FILE"
}

write_local_mysql_init_file() {
  cat >"$MYSQL_INIT_FILE" <<SQL
CREATE DATABASE IF NOT EXISTS \`$MYSQL_DATABASE\` CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
CREATE USER IF NOT EXISTS '$MYSQL_USER'@'127.0.0.1' IDENTIFIED BY '$MYSQL_PASSWORD';
CREATE USER IF NOT EXISTS '$MYSQL_USER'@'localhost' IDENTIFIED BY '$MYSQL_PASSWORD';
CREATE USER IF NOT EXISTS '$MYSQL_USER'@'%' IDENTIFIED BY '$MYSQL_PASSWORD';
GRANT ALL PRIVILEGES ON \`$MYSQL_DATABASE\`.* TO '$MYSQL_USER'@'127.0.0.1';
GRANT ALL PRIVILEGES ON \`$MYSQL_DATABASE\`.* TO '$MYSQL_USER'@'localhost';
GRANT ALL PRIVILEGES ON \`$MYSQL_DATABASE\`.* TO '$MYSQL_USER'@'%';
FLUSH PRIVILEGES;
SQL
  chmod 600 "$MYSQL_INIT_FILE"
}

start_native_mysql() {
  if [ ! -d "$MYSQL_DATA_DIR/mysql" ]; then
    need_cmd "mysqld" "$MYSQLD_BIN"
    if ! "$MYSQLD_BIN" --initialize-insecure --datadir="$MYSQL_DATA_DIR" > /dev/null 2>"$MYSQL_LOG_FILE"; then
      return 1
    fi
  fi

  write_local_mysql_init_file

  local mysql_start_bin="$MYSQLD_BIN"
  if [ -n "$MYSQLD_SAFE_BIN" ]; then
    mysql_start_bin="$MYSQLD_SAFE_BIN"
  fi

  nohup "$mysql_start_bin" \
    --datadir="$MYSQL_DATA_DIR" \
    --socket="$MYSQL_SOCKET" \
    --pid-file="$MYSQL_PID_FILE" \
    --port="$MYSQL_PORT" \
    --bind-address="$MYSQL_HOST" \
    --skip-networking=0 \
    --mysqlx=0 \
    --init-file="$MYSQL_INIT_FILE" \
    --log-error="$MYSQL_LOG_FILE" \
    >/dev/null 2>&1 < /dev/null &

  LOCAL_MYSQL_STARTED=1
  clear_mysql_docker_state()

  for _ in {1..60}; do
    if mysql_root_ping || mysql_app_ping; then
      break
    fi
    sleep 1
  done

  if ! mysql_root_ping && ! mysql_app_ping; then
    echo "MySQL failed to start. Check $MYSQL_LOG_FILE" >&2
    exit 1
  fi

  for _ in {1..30}; do
    if mysql_app_ping; then
      rm -f "$MYSQL_INIT_FILE"
      echo "Started local MySQL on ${MYSQL_HOST}:${MYSQL_PORT}"
      return
    fi
    sleep 1
  done

  echo "Local MySQL started but application user is not ready. Check $MYSQL_LOG_FILE" >&2
  exit 1
}

start_docker_mysql() {
  if ! docker_mysql_image_available; then
    return 1
  fi

  if docker_mysql_container_exists; then
    "$DOCKER_BIN" start "$MYSQL_DOCKER_CONTAINER" >/dev/null 2>&1 || return 1
  else
    "$DOCKER_BIN" run -d \
      --name "$MYSQL_DOCKER_CONTAINER" \
      -p "${MYSQL_PORT}:3306" \
      -e MYSQL_ROOT_PASSWORD="$MYSQL_ROOT_PASSWORD" \
      -e MYSQL_DATABASE="$MYSQL_DATABASE" \
      -e MYSQL_USER="$MYSQL_USER" \
      -e MYSQL_PASSWORD="$MYSQL_PASSWORD" \
      -v "$MYSQL_DATA_DIR:/var/lib/mysql" \
      "$MYSQL_DOCKER_IMAGE" >/dev/null || return 1
  fi

  LOCAL_MYSQL_STARTED=1
  mark_mysql_docker_state()

  for _ in {1..60}; do
    if mysql_root_ping || mysql_app_ping; then
      echo "Started Docker MySQL on ${MYSQL_HOST}:${MYSQL_PORT}"
      return 0
    fi
    sleep 1
  done

  echo "Docker MySQL failed to become ready." >&2
  exit 1
}

start_mysql() {
  if mysql_root_ping || mysql_app_ping; then
    clear_mysql_docker_state
    echo "Using existing MySQL on ${MYSQL_HOST}:${MYSQL_PORT}"
    return
  fi

  case "$MYSQL_RUNTIME_MODE" in
    docker)
      if start_docker_mysql; then
        return
      fi
      echo "Docker MySQL startup failed. Ensure Docker is running and image ${MYSQL_DOCKER_IMAGE} is available locally." >&2
      exit 1
      ;;
    native)
      if start_native_mysql; then
        return
      fi
      echo "Native MySQL startup failed. Check $MYSQL_LOG_FILE" >&2
      exit 1
      ;;
    auto)
      if [ -n "$MYSQLD_BIN" ] && start_native_mysql; then
        return
      fi
      if start_docker_mysql; then
        echo "Native MySQL startup failed, fell back to Docker image ${MYSQL_DOCKER_IMAGE}."
        return
      fi
      echo "MySQL startup failed in auto mode. Native bootstrap failed and Docker fallback is unavailable." >&2
      echo "Check $MYSQL_LOG_FILE, or set MYSQL_RUNTIME_MODE=docker with a local ${MYSQL_DOCKER_IMAGE} image." >&2
      exit 1
      ;;
    *)
      echo "Unsupported MYSQL_RUNTIME_MODE: ${MYSQL_RUNTIME_MODE}" >&2
      exit 1
      ;;
  esac
}

ensure_mysql_database() {
  if [ "$LOCAL_MYSQL_STARTED" -eq 1 ]; then
    if mysql_app_ping; then
      return
    fi
    echo "Local MySQL is running on ${MYSQL_HOST}:${MYSQL_PORT}, but ${MYSQL_DATABASE} is not accessible for ${MYSQL_USER}." >&2
    echo "Check $MYSQL_LOG_FILE for init-file execution errors." >&2
    exit 1
  fi

  if mysql_root_ping; then
  mysql_root_exec <<SQL
CREATE DATABASE IF NOT EXISTS \`$MYSQL_DATABASE\` CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
CREATE USER IF NOT EXISTS '$MYSQL_USER'@'127.0.0.1' IDENTIFIED BY '$MYSQL_PASSWORD';
CREATE USER IF NOT EXISTS '$MYSQL_USER'@'localhost' IDENTIFIED BY '$MYSQL_PASSWORD';
CREATE USER IF NOT EXISTS '$MYSQL_USER'@'%' IDENTIFIED BY '$MYSQL_PASSWORD';
GRANT ALL PRIVILEGES ON \`$MYSQL_DATABASE\`.* TO '$MYSQL_USER'@'127.0.0.1';
GRANT ALL PRIVILEGES ON \`$MYSQL_DATABASE\`.* TO '$MYSQL_USER'@'localhost';
GRANT ALL PRIVILEGES ON \`$MYSQL_DATABASE\`.* TO '$MYSQL_USER'@'%';
FLUSH PRIVILEGES;
SQL
    return
  fi

  if mysql_app_ping; then
    return
  fi

  echo "MySQL is reachable on ${MYSQL_HOST}:${MYSQL_PORT}, but neither root nor application credentials can access ${MYSQL_DATABASE}." >&2
  echo "Set MYSQL_DSN/MYSQL_* to a reachable database, or start a local MySQL instance with root access." >&2
  exit 1
}

start_redis() {
  if "$REDIS_CLI_BIN" -h "$REDIS_HOST" -p "$REDIS_PORT" ping >/dev/null 2>&1; then
    echo "Using existing Redis on ${REDIS_HOST}:${REDIS_PORT}"
    return
  fi

  "$REDIS_SERVER_BIN" \
    --port "$REDIS_PORT" \
    --bind "$REDIS_HOST" \
    --daemonize yes \
    --dir "$REDIS_DATA_DIR" \
    --pidfile "$REDIS_PID_FILE" \
    --logfile "$REDIS_LOG_FILE" \
    --save "" \
    --appendonly no

  for _ in {1..20}; do
    if "$REDIS_CLI_BIN" -h "$REDIS_HOST" -p "$REDIS_PORT" ping >/dev/null 2>&1; then
      break
    fi
    sleep 1
  done

  if ! "$REDIS_CLI_BIN" -h "$REDIS_HOST" -p "$REDIS_PORT" ping >/dev/null 2>&1; then
    echo "Redis failed to start. Check $REDIS_LOG_FILE" >&2
    exit 1
  fi

  echo "Started local Redis on ${REDIS_HOST}:${REDIS_PORT}"
}

run_migrations() {
  (
    cd "$BACKEND_DIR"
    env \
      MYSQL_DSN="$MYSQL_DSN" \
      REDIS_ADDR="$REDIS_ADDR" \
      RABBITMQ_URL="$RABBITMQ_URL" \
      ELASTICSEARCH_URL="$ELASTICSEARCH_URL" \
      JWT_SECRET="$JWT_SECRET" \
      go run ./cmd/migrate
  )
}

start_service() {
  local name="$1"
  local path="$2"
  local port="$3"
  local pid_file="$PID_DIR/${name}.pid"
  local log_file="$LOG_DIR/${name}.log"
  local binary="$BIN_DIR/${name}"

  if is_port_listening "$port"; then
    echo "Port ${port} already in use, skipping ${name}"
    return
  fi

  (
    cd "$BACKEND_DIR"
    env \
      APP_PORT="$port" \
      MYSQL_DSN="$MYSQL_DSN" \
      REDIS_ADDR="$REDIS_ADDR" \
      RABBITMQ_URL="$RABBITMQ_URL" \
      ELASTICSEARCH_URL="$ELASTICSEARCH_URL" \
      JWT_SECRET="$JWT_SECRET" \
      AUTH_SERVICE_URL="http://127.0.0.1:${AUTH_PORT}" \
      API_SERVICE_URL="http://127.0.0.1:${API_PORT}" \
      REALTIME_SERVICE_URL="http://127.0.0.1:${REALTIME_PORT}" \
      CORS_ORIGINS="$CORS_ORIGINS" \
      go build -o "$binary" "$path"
  )

  nohup bash -lc "cd '$BACKEND_DIR' && APP_PORT='$port' MYSQL_DSN='$MYSQL_DSN' REDIS_ADDR='$REDIS_ADDR' RABBITMQ_URL='$RABBITMQ_URL' ELASTICSEARCH_URL='$ELASTICSEARCH_URL' JWT_SECRET='$JWT_SECRET' AUTH_SERVICE_URL='http://127.0.0.1:${AUTH_PORT}' API_SERVICE_URL='http://127.0.0.1:${API_PORT}' REALTIME_SERVICE_URL='http://127.0.0.1:${REALTIME_PORT}' CORS_ORIGINS='$CORS_ORIGINS' '$binary'" >"$log_file" 2>&1 < /dev/null &

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

  if is_port_listening "$FRONTEND_PORT"; then
    echo "Port ${FRONTEND_PORT} already in use, skipping frontend"
    return
  fi

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

start_mysql
ensure_mysql_database
start_redis
run_migrations
start_service "auth-service" "./services/auth" "$AUTH_PORT"
start_service "api-service" "./services/api" "$API_PORT"
start_service "realtime-service" "./services/realtime" "$REALTIME_PORT"
start_frontend

cat <<EOF
Local enterprise workspace is up.
Auth:      http://127.0.0.1:${AUTH_PORT}
API:       http://127.0.0.1:${API_PORT}
Realtime:  ws://127.0.0.1:${REALTIME_PORT}/ws
Frontend:  http://127.0.0.1:${FRONTEND_PORT}
Runtime:   ${RUNTIME_DIR}
EOF

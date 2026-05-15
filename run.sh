#!/usr/bin/env sh

set -eu

ACTION="${1:-start}"
APP_PORT="${APP_PORT:-8080}"
RUNTIME_DIR="${RUNTIME_DIR:-$HOME/.cache/ezbookkeeping-runtime}"
BIN_PATH="${BIN_PATH:-$RUNTIME_DIR/ezbookkeeping}"
BUILD_BACKEND="${BUILD_BACKEND:-1}"
BUILD_FRONTEND="${BUILD_FRONTEND:-1}"
STOP_DOCKER_COMPOSE="${STOP_DOCKER_COMPOSE:-1}"
STOP_TIMEOUT_SECONDS="${STOP_TIMEOUT_SECONDS:-15}"
HEALTH_TIMEOUT_SECONDS="${HEALTH_TIMEOUT_SECONDS:-30}"
USE_LAUNCHD="${USE_LAUNCHD:-auto}"
LAUNCHD_LABEL="${LAUNCHD_LABEL:-ezbookkeeping-codex}"
LAUNCHD_SERVICE="gui/$(id -u)/$LAUNCHD_LABEL"
PID_FILE="$RUNTIME_DIR/server.pid"
LOG_FILE="$RUNTIME_DIR/server.log"
ERR_LOG_FILE="$RUNTIME_DIR/server.err.log"

echo_red() {
    printf '\033[31m%s\033[0m\n' "$1"
}

print_go_install_help() {
    echo_red "Error: \"go\" is required to build the backend."
    echo ""
    echo "This is a frontend + backend project:"
    echo "  - Go + GCC are used to build the backend binary"
    echo "  - Node.js + npm are used to build the frontend files"
    echo ""
    echo "Install Go first, then run npm start again. On macOS with Homebrew:"
    echo "  brew install go"
    echo ""
    echo "Temporary fallback only, if you intentionally want to reuse an existing backend binary:"
    echo "  BUILD_BACKEND=0 npm start"
}

show_help() {
    cat <<-EOF
ezBookkeeping local run script

Usage:
    ./run.sh [start|stop|restart|status]

Aliases:
    run      Same as start
    pause    Same as stop

Environment:
    APP_PORT=8080                         HTTP port
    RUNTIME_DIR=~/.cache/ezbookkeeping-runtime
    BIN_PATH=\$RUNTIME_DIR/ezbookkeeping
    BUILD_BACKEND=1                       Build backend before start, set 0 to reuse BIN_PATH
    BUILD_FRONTEND=1                      Build frontend before start
    STOP_DOCKER_COMPOSE=1                 Stop local compose app before start/stop
    USE_LAUNCHD=auto                      Use launchctl on macOS, set 0 to use nohup
EOF
}

normalize_action() {
    case "$ACTION" in
        start | run)
            ACTION="start"
            ;;
        stop | pause)
            ACTION="stop"
            ;;
        restart | status)
            ;;
        help | -h | --help)
            show_help
            exit 0
            ;;
        *)
            echo_red "Invalid action: $ACTION"
            show_help
            exit 2
            ;;
    esac
}

check_dependency() {
    for cmd in $1
    do
        if ! command -v "$cmd" >/dev/null 2>&1; then
            echo_red "Error: \"$cmd\" is required."
            exit 127
        fi
    done
}

ensure_runtime_dirs() {
    mkdir -p "$RUNTIME_DIR/data" "$RUNTIME_DIR/log" "$RUNTIME_DIR/storage"
}

process_command() {
    ps -p "$1" -o command= 2>/dev/null || true
}

is_ezbookkeeping_process() {
    COMMAND="$(process_command "$1")"

    case "$COMMAND" in
        *ezbookkeeping*)
            return 0
            ;;
        *)
            return 1
            ;;
    esac
}

port_listen_pids() {
    if command -v lsof >/dev/null 2>&1; then
        lsof -tiTCP:"$APP_PORT" -sTCP:LISTEN 2>/dev/null || true
    fi
}

use_launchd() {
    if [ "$USE_LAUNCHD" = "0" ]; then
        return 1
    fi

    if command -v launchctl >/dev/null 2>&1 && [ "$(uname -s 2>/dev/null || true)" = "Darwin" ]; then
        return 0
    fi

    return 1
}

launchd_job_exists() {
    use_launchd && launchctl print "$LAUNCHD_SERVICE" >/dev/null 2>&1
}

remove_launchd_job() {
    if ! use_launchd; then
        return 1
    fi

    if launchd_job_exists; then
        launchctl remove "$LAUNCHD_LABEL" >/dev/null 2>&1 || true
        return 0
    fi

    return 1
}

stop_docker_compose_app() {
    if [ "$STOP_DOCKER_COMPOSE" != "1" ] || ! command -v docker >/dev/null 2>&1; then
        return 0
    fi

    COMPOSE_IDS="$(docker compose ps -q app 2>/dev/null || true)"

    if [ -n "$COMPOSE_IDS" ]; then
        echo "Stopping Docker Compose app service..."
        docker compose stop app >/dev/null
    fi
}

stop_pid() {
    PID="$1"

    if [ -z "$PID" ] || ! kill -0 "$PID" 2>/dev/null; then
        return 1
    fi

    if ! is_ezbookkeeping_process "$PID"; then
        COMMAND="$(process_command "$PID")"
        echo "Process $PID is not ezBookkeeping: $COMMAND"
        return 2
    fi

    echo "Stopping ezBookkeeping process $PID..."
    kill "$PID" 2>/dev/null || true
    return 0
}

force_stop_pid() {
    PID="$1"

    if [ -z "$PID" ] || ! kill -0 "$PID" 2>/dev/null; then
        return 1
    fi

    if ! is_ezbookkeeping_process "$PID"; then
        COMMAND="$(process_command "$PID")"
        echo "Process $PID is still not ezBookkeeping: $COMMAND"
        return 2
    fi

    echo "Force stopping ezBookkeeping process $PID..."
    kill -9 "$PID" 2>/dev/null || true
    return 0
}

wait_until_port_is_free() {
    WAITED=0

    while [ "$WAITED" -lt "$STOP_TIMEOUT_SECONDS" ]; do
        PIDS="$(port_listen_pids)"

        if [ -z "$PIDS" ]; then
            return 0
        fi

        sleep 1
        WAITED=$((WAITED + 1))
    done

    return 1
}

stop_port_processes() {
    PIDS="$(port_listen_pids)"

    if [ -z "$PIDS" ]; then
        return 1
    fi

    for PID in $PIDS; do
        if ! stop_pid "$PID"; then
            COMMAND="$(process_command "$PID")"
            echo "Port $APP_PORT is used by another process: $COMMAND"
            echo "Please stop it manually or set APP_PORT to another port."
            exit 1
        fi
    done

    if ! wait_until_port_is_free; then
        PIDS="$(port_listen_pids)"

        for PID in $PIDS; do
            if ! force_stop_pid "$PID"; then
                COMMAND="$(process_command "$PID")"
                echo "Port $APP_PORT is still used by another process: $COMMAND"
                echo "Please stop it manually or set APP_PORT to another port."
                exit 1
            fi
        done
    fi

    return 0
}

stop_app() {
    ensure_runtime_dirs
    STOPPED="0"

    if remove_launchd_job; then
        echo "Stopping launchd service $LAUNCHD_LABEL..."
        STOPPED="1"
    fi

    if [ -f "$PID_FILE" ]; then
        PID="$(sed -n '1p' "$PID_FILE" 2>/dev/null || true)"

        if [ -n "$PID" ] && kill -0 "$PID" 2>/dev/null; then
            if stop_pid "$PID"; then
                STOPPED="1"
            fi
        fi
    fi

    if stop_port_processes; then
        STOPPED="1"
    fi

    stop_docker_compose_app

    if [ "$STOPPED" = "1" ]; then
        if ! wait_until_port_is_free; then
            PIDS="$(port_listen_pids)"

            for PID in $PIDS; do
                force_stop_pid "$PID" >/dev/null 2>&1 || true
            done
        fi

        : > "$PID_FILE"
        echo "ezBookkeeping has stopped."
    else
        : > "$PID_FILE"
        echo "ezBookkeeping is not running."
    fi
}

build_backend() {
    if [ "$BUILD_BACKEND" = "0" ] && [ -x "$BIN_PATH" ]; then
        return 0
    fi

    if ! command -v go >/dev/null 2>&1; then
        print_go_install_help
        exit 127
    fi

    check_dependency "gcc"
    ensure_runtime_dirs

    echo "Building backend binary..."
    go build -o "$BIN_PATH" ezbookkeeping.go
    chmod +x "$BIN_PATH"
}

build_frontend() {
    if [ "$BUILD_FRONTEND" != "1" ]; then
        return 0
    fi

    check_dependency "node npm"

    echo "Building frontend files..."
    npm run build
}

record_listen_pid() {
    PIDS="$(port_listen_pids)"

    for PID in $PIDS; do
        if is_ezbookkeeping_process "$PID"; then
            echo "$PID" > "$PID_FILE"
            return 0
        fi
    done

    return 1
}

wait_for_health() {
    WAITED=0

    while [ "$WAITED" -lt "$HEALTH_TIMEOUT_SECONDS" ]; do
        if command -v curl >/dev/null 2>&1 && curl -fsS "http://127.0.0.1:$APP_PORT/healthz.json" >/dev/null 2>&1; then
            if record_listen_pid; then
                sleep 1

                if curl -fsS "http://127.0.0.1:$APP_PORT/healthz.json" >/dev/null 2>&1; then
                    return 0
                fi
            fi
        fi

        sleep 1
        WAITED=$((WAITED + 1))
    done

    return 1
}

start_service_process() {
    if use_launchd; then
        remove_launchd_job >/dev/null 2>&1 || true
        echo "Starting ezBookkeeping service with launchctl..."
        launchctl submit \
            -l "$LAUNCHD_LABEL" \
            -o "$LOG_FILE" \
            -e "$ERR_LOG_FILE" \
            -- /usr/bin/env \
            EBK_SERVER_HTTP_PORT="$APP_PORT" \
            EBK_WORK_DIR="${EBK_WORK_DIR:-$PWD}" \
            EBK_SERVER_DOMAIN="${EBK_SERVER_DOMAIN:-localhost}" \
            EBK_SERVER_ROOT_URL="${EBK_SERVER_ROOT_URL:-http://localhost:$APP_PORT/}" \
            EBK_SERVER_STATIC_ROOT_PATH="${EBK_SERVER_STATIC_ROOT_PATH:-$PWD/dist}" \
            EBK_DATABASE_DB_PATH="${EBK_DATABASE_DB_PATH:-$RUNTIME_DIR/data/ezbookkeeping.db}" \
            EBK_LOG_MODE="${EBK_LOG_MODE:-console}" \
            EBK_STORAGE_LOCAL_FILESYSTEM_PATH="${EBK_STORAGE_LOCAL_FILESYSTEM_PATH:-$RUNTIME_DIR/storage}" \
            EBK_SECURITY_SECRET_KEY="${EBK_SECURITY_SECRET_KEY:-change-me-before-production}" \
            "$BIN_PATH" --conf-path "${EBK_CONF_PATH:-$PWD/conf/ezbookkeeping.ini}" server run
        return 0
    fi

    echo "Starting ezBookkeeping service..."
    EBK_SERVER_HTTP_PORT="$APP_PORT" \
    EBK_WORK_DIR="${EBK_WORK_DIR:-$PWD}" \
    EBK_SERVER_DOMAIN="${EBK_SERVER_DOMAIN:-localhost}" \
    EBK_SERVER_ROOT_URL="${EBK_SERVER_ROOT_URL:-http://localhost:$APP_PORT/}" \
    EBK_SERVER_STATIC_ROOT_PATH="${EBK_SERVER_STATIC_ROOT_PATH:-$PWD/dist}" \
    EBK_DATABASE_DB_PATH="${EBK_DATABASE_DB_PATH:-$RUNTIME_DIR/data/ezbookkeeping.db}" \
    EBK_LOG_MODE="${EBK_LOG_MODE:-console}" \
    EBK_STORAGE_LOCAL_FILESYSTEM_PATH="${EBK_STORAGE_LOCAL_FILESYSTEM_PATH:-$RUNTIME_DIR/storage}" \
    EBK_SECURITY_SECRET_KEY="${EBK_SECURITY_SECRET_KEY:-change-me-before-production}" \
    nohup "$BIN_PATH" --conf-path "${EBK_CONF_PATH:-$PWD/conf/ezbookkeeping.ini}" server run > "$LOG_FILE" 2> "$ERR_LOG_FILE" < /dev/null &
    NEW_PID="$!"

    echo "$NEW_PID" > "$PID_FILE"
}

show_error_log() {
    echo "Error log:"
    tail -50 "$ERR_LOG_FILE" 2>/dev/null || true
}

start_app() {
    build_backend
    build_frontend
    ensure_runtime_dirs
    stop_docker_compose_app
    stop_port_processes || true

    if ! [ -x "$BIN_PATH" ]; then
        echo_red "Cannot find executable ezBookkeeping binary: $BIN_PATH"
        exit 1
    fi

    : > "$LOG_FILE"
    : > "$ERR_LOG_FILE"

    start_service_process

    if ! wait_for_health; then
        echo "ezBookkeeping failed to start or health check failed."
        show_error_log
        exit 1
    fi

    echo "ezBookkeeping is running at http://localhost:${APP_PORT}/"
    echo "Logs: $LOG_FILE"
}

status_app() {
    if [ -f "$PID_FILE" ]; then
        PID="$(sed -n '1p' "$PID_FILE" 2>/dev/null || true)"

        if [ -n "$PID" ] && kill -0 "$PID" 2>/dev/null && is_ezbookkeeping_process "$PID"; then
            echo "ezBookkeeping is running with pid $PID at http://localhost:${APP_PORT}/"
            exit 0
        fi
    fi

    PIDS="$(port_listen_pids)"

    for PID in $PIDS; do
        if is_ezbookkeeping_process "$PID"; then
            echo "$PID" > "$PID_FILE"
            echo "ezBookkeeping is running with pid $PID at http://localhost:${APP_PORT}/"
            exit 0
        fi
    done

    echo "ezBookkeeping is not running."
    exit 3
}

normalize_action

case "$ACTION" in
    start)
        start_app
        ;;
    stop)
        stop_app
        ;;
    restart)
        stop_app
        start_app
        ;;
    status)
        status_app
        ;;
esac

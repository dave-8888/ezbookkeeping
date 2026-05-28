#!/usr/bin/env sh

set -eu

SCRIPT_DIR="$(CDPATH= cd "$(dirname "$0")" && pwd)"
cd "$SCRIPT_DIR"

ACTION="${1:-start}"
APP_PORT="${APP_PORT:-8888}"
FRONTEND_PORT="${FRONTEND_PORT:-8889}"
FRONTEND_HOST="${FRONTEND_HOST:-0.0.0.0}"
FRONTEND_PORTS="${FRONTEND_PORTS:-$FRONTEND_PORT 8081 5173 4173}"
RUNTIME_DIR="${RUNTIME_DIR:-$HOME/.cache/financially_free-runtime}"
BIN_PATH="${BIN_PATH:-$RUNTIME_DIR/financially_free}"
BUILD_BACKEND="${BUILD_BACKEND:-1}"
BUILD_FRONTEND="${BUILD_FRONTEND:-1}"
START_FRONTEND="${START_FRONTEND:-1}"
STOP_DOCKER_COMPOSE="${STOP_DOCKER_COMPOSE:-1}"
STOP_TIMEOUT_SECONDS="${STOP_TIMEOUT_SECONDS:-15}"
HEALTH_TIMEOUT_SECONDS="${HEALTH_TIMEOUT_SECONDS:-30}"
USE_LAUNCHD="${USE_LAUNCHD:-auto}"
LAUNCHD_LABEL="${LAUNCHD_LABEL:-financially_free-codex}"
FRONTEND_LAUNCHD_LABEL="${FRONTEND_LAUNCHD_LABEL:-financially_free-frontend-codex}"
LEGACY_RUNTIME_DIR="${LEGACY_RUNTIME_DIR:-$HOME/.cache/ezbookkeeping-runtime}"
LEGACY_BIN_PATH="${LEGACY_BIN_PATH:-$LEGACY_RUNTIME_DIR/ezbookkeeping}"
LEGACY_CONF_PATH="${LEGACY_CONF_PATH:-$PWD/conf/ezbookkeeping.ini}"
LEGACY_LAUNCHD_LABEL="${LEGACY_LAUNCHD_LABEL:-ezbookkeeping-codex}"
LEGACY_FRONTEND_LAUNCHD_LABEL="${LEGACY_FRONTEND_LAUNCHD_LABEL:-ezbookkeeping-frontend-codex}"
LAUNCHD_SERVICE="gui/$(id -u)/$LAUNCHD_LABEL"
FRONTEND_LAUNCHD_SERVICE="gui/$(id -u)/$FRONTEND_LAUNCHD_LABEL"
LEGACY_LAUNCHD_SERVICE="gui/$(id -u)/$LEGACY_LAUNCHD_LABEL"
LEGACY_FRONTEND_LAUNCHD_SERVICE="gui/$(id -u)/$LEGACY_FRONTEND_LAUNCHD_LABEL"
PID_FILE="$RUNTIME_DIR/server.pid"
LOG_FILE="$RUNTIME_DIR/server.log"
ERR_LOG_FILE="$RUNTIME_DIR/server.err.log"
FRONTEND_PID_FILE="$RUNTIME_DIR/frontend.pid"
FRONTEND_LOG_FILE="$RUNTIME_DIR/frontend.log"
FRONTEND_ERR_LOG_FILE="$RUNTIME_DIR/frontend.err.log"

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
financially_free local run script

Usage:
    ./run.sh [start|stop|restart|status]

Aliases:
    run      Same as start
    pause    Same as stop

Environment:
    APP_PORT=8888                         Backend HTTP port
    FRONTEND_PORT=8889                    Vite dev server port
    FRONTEND_HOST=0.0.0.0                 Vite dev server host
    FRONTEND_PORTS="8889 8081 5173 4173"  Vite dev/preview ports to stop for this project
    RUNTIME_DIR=~/.cache/financially_free-runtime
    BIN_PATH=\$RUNTIME_DIR/financially_free
    BUILD_BACKEND=1                       Build backend before start, set 0 to reuse BIN_PATH
    BUILD_FRONTEND=1                      Build frontend before start
    START_FRONTEND=1                      Start Vite dev server after backend is healthy
    STOP_DOCKER_COMPOSE=1                 Stop local compose app before start/stop
    USE_LAUNCHD=auto                      Use launchctl on macOS, set 0 to use nohup
    LAUNCHD_LABEL=financially_free-codex
    FRONTEND_LAUNCHD_LABEL=financially_free-frontend-codex
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

process_cwd() {
    if command -v lsof >/dev/null 2>&1; then
        lsof -a -p "$1" -d cwd -Fn 2>/dev/null | sed -n 's/^n//p' | head -1
    fi
}

is_financially_free_process() {
    COMMAND="$(process_command "$1")"

    case "$COMMAND" in
        *financially_free*)
            return 0
            ;;
    esac

    case "$COMMAND" in
        *"$LEGACY_BIN_PATH"* | *"$LEGACY_CONF_PATH"*)
            return 0
            ;;
    esac

    return 1
}

port_listen_pids() {
    if command -v lsof >/dev/null 2>&1; then
        lsof -tiTCP:"$APP_PORT" -sTCP:LISTEN 2>/dev/null || true
    fi
}

frontend_listen_pids() {
    if command -v lsof >/dev/null 2>&1; then
        for PORT in $FRONTEND_PORTS; do
            lsof -tiTCP:"$PORT" -sTCP:LISTEN 2>/dev/null || true
        done | sort -u
    fi
}

is_current_project_frontend_process() {
    COMMAND="$(process_command "$1")"
    CWD="$(process_cwd "$1")"

    if [ "$CWD" = "$PWD" ]; then
        case "$COMMAND" in
            *node* | *npm* | *vite*)
                return 0
                ;;
        esac
    fi

    case "$COMMAND" in
        *"$PWD"*)
            case "$COMMAND" in
                *node* | *npm* | *vite*)
                    return 0
                    ;;
            esac
            ;;
    esac

    return 1
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

legacy_launchd_job_exists() {
    use_launchd && launchctl print "$LEGACY_LAUNCHD_SERVICE" >/dev/null 2>&1
}

remove_legacy_launchd_job() {
    if ! use_launchd; then
        return 1
    fi

    if legacy_launchd_job_exists; then
        launchctl remove "$LEGACY_LAUNCHD_LABEL" >/dev/null 2>&1 || true
        return 0
    fi

    return 1
}

frontend_launchd_job_exists() {
    use_launchd && launchctl print "$FRONTEND_LAUNCHD_SERVICE" >/dev/null 2>&1
}

remove_frontend_launchd_job() {
    if ! use_launchd; then
        return 1
    fi

    if frontend_launchd_job_exists; then
        launchctl remove "$FRONTEND_LAUNCHD_LABEL" >/dev/null 2>&1 || true
        return 0
    fi

    return 1
}

legacy_frontend_launchd_job_exists() {
    use_launchd && launchctl print "$LEGACY_FRONTEND_LAUNCHD_SERVICE" >/dev/null 2>&1
}

remove_legacy_frontend_launchd_job() {
    if ! use_launchd; then
        return 1
    fi

    if legacy_frontend_launchd_job_exists; then
        launchctl remove "$LEGACY_FRONTEND_LAUNCHD_LABEL" >/dev/null 2>&1 || true
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

    if ! is_financially_free_process "$PID"; then
        COMMAND="$(process_command "$PID")"
        echo "Process $PID is not financially_free: $COMMAND"
        return 2
    fi

    echo "Stopping financially_free process $PID..."
    kill "$PID" 2>/dev/null || true
    return 0
}

force_stop_pid() {
    PID="$1"

    if [ -z "$PID" ] || ! kill -0 "$PID" 2>/dev/null; then
        return 1
    fi

    if ! is_financially_free_process "$PID"; then
        COMMAND="$(process_command "$PID")"
        echo "Process $PID is still not financially_free: $COMMAND"
        return 2
    fi

    echo "Force stopping financially_free process $PID..."
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

stop_frontend_dev_processes() {
    PIDS="$(frontend_listen_pids)"

    if [ -z "$PIDS" ]; then
        return 1
    fi

    STOPPED_FRONTEND="0"

    for PID in $PIDS; do
        if is_current_project_frontend_process "$PID"; then
            echo "Stopping financially_free frontend dev process $PID..."
            kill "$PID" 2>/dev/null || true
            STOPPED_FRONTEND="1"
        fi
    done

    if [ "$STOPPED_FRONTEND" != "1" ]; then
        return 1
    fi

    WAITED=0

    while [ "$WAITED" -lt "$STOP_TIMEOUT_SECONDS" ]; do
        REMAINING=""

        for PID in $(frontend_listen_pids); do
            if is_current_project_frontend_process "$PID"; then
                REMAINING="$REMAINING $PID"
            fi
        done

        if [ -z "$REMAINING" ]; then
            return 0
        fi

        sleep 1
        WAITED=$((WAITED + 1))
    done

    for PID in $(frontend_listen_pids); do
        if is_current_project_frontend_process "$PID"; then
            echo "Force stopping financially_free frontend dev process $PID..."
            kill -9 "$PID" 2>/dev/null || true
        fi
    done

    return 0
}

stop_app() {
    ensure_runtime_dirs
    STOPPED="0"

    if remove_launchd_job; then
        echo "Stopping launchd service $LAUNCHD_LABEL..."
        STOPPED="1"
    fi

    if remove_legacy_launchd_job; then
        echo "Stopping launchd service $LEGACY_LAUNCHD_LABEL..."
        STOPPED="1"
    fi

    if remove_frontend_launchd_job; then
        echo "Stopping launchd service $FRONTEND_LAUNCHD_LABEL..."
        STOPPED="1"
    fi

    if remove_legacy_frontend_launchd_job; then
        echo "Stopping launchd service $LEGACY_FRONTEND_LAUNCHD_LABEL..."
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

    if stop_frontend_dev_processes; then
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
        : > "$FRONTEND_PID_FILE"
        echo "financially_free has stopped."
    else
        : > "$PID_FILE"
        : > "$FRONTEND_PID_FILE"
        echo "financially_free is not running."
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
    go build -o "$BIN_PATH" financially_free.go
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
        if is_financially_free_process "$PID"; then
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

wait_for_frontend() {
    WAITED=0

    while [ "$WAITED" -lt "$HEALTH_TIMEOUT_SECONDS" ]; do
        if command -v curl >/dev/null 2>&1 && curl -fsS "http://127.0.0.1:$FRONTEND_PORT/" >/dev/null 2>&1; then
            return 0
        fi

        if [ -f "$FRONTEND_PID_FILE" ]; then
            PID="$(sed -n '1p' "$FRONTEND_PID_FILE" 2>/dev/null || true)"

            if [ -n "$PID" ] && ! kill -0 "$PID" 2>/dev/null; then
                return 1
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
        echo "Starting financially_free service with launchctl..."
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
            EBK_DATABASE_DB_PATH="${EBK_DATABASE_DB_PATH:-$RUNTIME_DIR/data/financially_free.db}" \
            EBK_LOG_MODE="${EBK_LOG_MODE:-console}" \
            EBK_STORAGE_LOCAL_FILESYSTEM_PATH="${EBK_STORAGE_LOCAL_FILESYSTEM_PATH:-$RUNTIME_DIR/storage}" \
            EBK_SECURITY_SECRET_KEY="${EBK_SECURITY_SECRET_KEY:-change-me-before-production}" \
            "$BIN_PATH" --conf-path "${EBK_CONF_PATH:-$PWD/conf/financially_free.ini}" server run
        return 0
    fi

    echo "Starting financially_free service..."
    EBK_SERVER_HTTP_PORT="$APP_PORT" \
    EBK_WORK_DIR="${EBK_WORK_DIR:-$PWD}" \
    EBK_SERVER_DOMAIN="${EBK_SERVER_DOMAIN:-localhost}" \
    EBK_SERVER_ROOT_URL="${EBK_SERVER_ROOT_URL:-http://localhost:$APP_PORT/}" \
    EBK_SERVER_STATIC_ROOT_PATH="${EBK_SERVER_STATIC_ROOT_PATH:-$PWD/dist}" \
    EBK_DATABASE_DB_PATH="${EBK_DATABASE_DB_PATH:-$RUNTIME_DIR/data/financially_free.db}" \
    EBK_LOG_MODE="${EBK_LOG_MODE:-console}" \
    EBK_STORAGE_LOCAL_FILESYSTEM_PATH="${EBK_STORAGE_LOCAL_FILESYSTEM_PATH:-$RUNTIME_DIR/storage}" \
    EBK_SECURITY_SECRET_KEY="${EBK_SECURITY_SECRET_KEY:-change-me-before-production}" \
    nohup "$BIN_PATH" --conf-path "${EBK_CONF_PATH:-$PWD/conf/financially_free.ini}" server run > "$LOG_FILE" 2> "$ERR_LOG_FILE" < /dev/null &
    NEW_PID="$!"

    echo "$NEW_PID" > "$PID_FILE"
}

show_error_log() {
    echo "Error log:"
    tail -50 "$ERR_LOG_FILE" 2>/dev/null || true
}

show_frontend_error_log() {
    echo "Frontend error log:"
    tail -50 "$FRONTEND_ERR_LOG_FILE" 2>/dev/null || true
}

start_frontend_dev() {
    if [ "$START_FRONTEND" != "1" ]; then
        return 0
    fi

    check_dependency "node npm"
    NPM_PATH="$(command -v npm)"
    stop_frontend_dev_processes || true
    remove_frontend_launchd_job >/dev/null 2>&1 || true
    remove_legacy_frontend_launchd_job >/dev/null 2>&1 || true

    : > "$FRONTEND_LOG_FILE"
    : > "$FRONTEND_ERR_LOG_FILE"
    : > "$FRONTEND_PID_FILE"

    if use_launchd; then
        echo "Starting financially_free frontend dev server with launchctl..."
        launchctl submit \
            -l "$FRONTEND_LAUNCHD_LABEL" \
            -o "$FRONTEND_LOG_FILE" \
            -e "$FRONTEND_ERR_LOG_FILE" \
            -- /usr/bin/env \
            PATH="$PATH" \
            APP_PORT="$APP_PORT" \
            BACKEND_PORT="$APP_PORT" \
            FRONTEND_PORT="$FRONTEND_PORT" \
            FRONTEND_HOST="$FRONTEND_HOST" \
            EBK_FRONTEND_CWD="$PWD" \
            EBK_NPM_PATH="$NPM_PATH" \
            /bin/sh -lc 'cd "$EBK_FRONTEND_CWD" && exec "$EBK_NPM_PATH" run serve -- --host "$FRONTEND_HOST" --port "$FRONTEND_PORT"'
    else
        echo "Starting financially_free frontend dev server..."
        APP_PORT="$APP_PORT" \
        BACKEND_PORT="$APP_PORT" \
        FRONTEND_PORT="$FRONTEND_PORT" \
        FRONTEND_HOST="$FRONTEND_HOST" \
        nohup "$NPM_PATH" run serve -- --host "$FRONTEND_HOST" --port "$FRONTEND_PORT" > "$FRONTEND_LOG_FILE" 2> "$FRONTEND_ERR_LOG_FILE" < /dev/null &
        NEW_FRONTEND_PID="$!"

        echo "$NEW_FRONTEND_PID" > "$FRONTEND_PID_FILE"
    fi

    if ! wait_for_frontend; then
        echo "financially_free frontend failed to start or health check failed."
        show_frontend_error_log
        exit 1
    fi

    echo "financially_free frontend is running at http://localhost:${FRONTEND_PORT}/"
    echo "Frontend logs: $FRONTEND_LOG_FILE"
}

start_app() {
    build_backend
    build_frontend
    ensure_runtime_dirs
    stop_docker_compose_app
    remove_legacy_launchd_job >/dev/null 2>&1 || true
    stop_port_processes || true

    if ! [ -x "$BIN_PATH" ]; then
        echo_red "Cannot find executable financially_free binary: $BIN_PATH"
        exit 1
    fi

    : > "$LOG_FILE"
    : > "$ERR_LOG_FILE"

    start_service_process

    if ! wait_for_health; then
        echo "financially_free failed to start or health check failed."
        show_error_log
        exit 1
    fi

    echo "financially_free is running at http://localhost:${APP_PORT}/"
    echo "Logs: $LOG_FILE"
    start_frontend_dev
}

status_app() {
    RUNNING="0"

    if [ -f "$PID_FILE" ]; then
        PID="$(sed -n '1p' "$PID_FILE" 2>/dev/null || true)"

        if [ -n "$PID" ] && kill -0 "$PID" 2>/dev/null && is_financially_free_process "$PID"; then
            echo "financially_free is running with pid $PID at http://localhost:${APP_PORT}/"
            RUNNING="1"
        fi
    fi

    PIDS="$(port_listen_pids)"

    for PID in $PIDS; do
        if is_financially_free_process "$PID"; then
            echo "$PID" > "$PID_FILE"
            if [ "$RUNNING" != "1" ]; then
                echo "financially_free is running with pid $PID at http://localhost:${APP_PORT}/"
            fi
            RUNNING="1"
            break
        fi
    done

    if [ "$RUNNING" != "1" ] && [ -n "$PIDS" ]; then
        for PID in $PIDS; do
            COMMAND="$(process_command "$PID")"
            echo "Port $APP_PORT is occupied by another process: $COMMAND"
        done
    fi

    for PID in $(frontend_listen_pids); do
        if is_current_project_frontend_process "$PID"; then
            echo "financially_free frontend dev server is running with pid $PID."
            RUNNING="1"
        fi
    done

    if [ "$RUNNING" = "1" ]; then
        exit 0
    fi

    echo "financially_free is not running."
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

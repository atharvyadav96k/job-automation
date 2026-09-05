#!/bin/bash
# Runs the full stack without Docker, using the local Postgres/Redis/Go/Node
# installs under ~/.local (set up because this sandbox couldn't reach the
# Docker daemon — see plan.md's docker-compose setup for the normal path).
# All config comes from .env at the repo root — nothing is hardcoded here.
#
# Usage:
#   scripts/run-local.sh start   # start everything (default)
#   scripts/run-local.sh stop    # stop everything
#   scripts/run-local.sh status  # show what's running
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
ENV_FILE="$REPO_ROOT/.env"
[ -f "$ENV_FILE" ] || {
	echo "missing $ENV_FILE — copy .env.example to .env and fill it in first" >&2
	exit 1
}
set -a
# shellcheck disable=SC1090
source "$ENV_FILE"
set +a

PG_BIN="$HOME/.local/pgredis/usr/lib/postgresql/16/bin"
PG_DATA="$HOME/.local/pgdata"
PG_LIB="$HOME/.local/pgredis/usr/lib/x86_64-linux-gnu"
REDIS_BIN="$HOME/.local/pgredis/usr/bin"
LOG_DIR="$HOME/.local/job-automation-logs"
PID_DIR="$HOME/.local/job-automation-pids"
mkdir -p "$LOG_DIR" "$PID_DIR"

export PATH="$HOME/.local/node/bin:$HOME/.local/go/bin:$PATH"
export LD_LIBRARY_PATH="$PG_LIB:${LD_LIBRARY_PATH:-}"

# Parsed straight out of DATABASE_URL/REDIS_URL from .env, so the port
# postgres/redis actually listen on always matches what the app is told.
pg_port="$(echo "$DATABASE_URL" | sed -E 's#.*:([0-9]+)/.*#\1#')"
redis_port="$(echo "$REDIS_URL" | sed -E 's#.*:([0-9]+).*#\1#')"
frontend_port="$(echo "${FRONTEND_ORIGIN:-http://localhost:3000}" | sed -E 's#.*:([0-9]+).*#\1#')"

action="${1:-start}"

is_running() { # is_running <pidfile>
	[ -f "$1" ] && kill -0 "$(cat "$1")" 2>/dev/null
}

stop_all() {
	for name in server frontend; do
		pidfile="$PID_DIR/$name.pid"
		if is_running "$pidfile"; then
			# kill the whole process group, not just the launcher PID — `go
			# run` and `npx vite` both spawn a child that outlives the
			# parent if only the parent is signaled.
			pkill -P "$(cat "$pidfile")" 2>/dev/null || true
			kill "$(cat "$pidfile")" 2>/dev/null || true
			rm -f "$pidfile"
			echo "stopped $name"
		fi
	done
	pkill -f "go-build.*/exe/server" 2>/dev/null || true
	pkill -f "node_modules/.bin/vite --port $frontend_port" 2>/dev/null || true
	if "$PG_BIN/pg_ctl" -D "$PG_DATA" status >/dev/null 2>&1; then
		"$PG_BIN/pg_ctl" -D "$PG_DATA" stop -m fast >/dev/null
		echo "stopped postgres"
	fi
	# redis renames its own argv to "redis-server *:<port>", so shut it down
	# via its own protocol rather than pattern-matching the launch command.
	if "$REDIS_BIN/redis-cli" -p "$redis_port" shutdown nosave 2>/dev/null; then
		echo "stopped redis"
	fi
}

status_all() {
	"$PG_BIN/pg_ctl" -D "$PG_DATA" status 2>&1 || true
	"$REDIS_BIN/redis-cli" -p "$redis_port" ping 2>&1 || echo "redis: down"
	for name in server frontend; do
		pidfile="$PID_DIR/$name.pid"
		if is_running "$pidfile"; then
			echo "$name: running (pid $(cat "$pidfile"))"
		else
			echo "$name: down"
		fi
	done
}

start_all() {
	if ! "$PG_BIN/pg_ctl" -D "$PG_DATA" status >/dev/null 2>&1; then
		if [ ! -d "$PG_DATA" ]; then
			"$PG_BIN/initdb" -D "$PG_DATA" -U "${POSTGRES_USER:-jobauto}" --auth=trust >/dev/null
		fi
		"$PG_BIN/pg_ctl" -D "$PG_DATA" -l "$LOG_DIR/postgres.log" -o "-p $pg_port -k /tmp" start
		"$PG_BIN/createdb" -h 127.0.0.1 -p "$pg_port" -U "${POSTGRES_USER:-jobauto}" "${POSTGRES_DB:-jobauto}" 2>/dev/null || true
	fi

	if ! "$REDIS_BIN/redis-cli" -p "$redis_port" ping >/dev/null 2>&1; then
		"$REDIS_BIN/redis-server" --port "$redis_port" --dir "$HOME/.local" --logfile "$LOG_DIR/redis.log" --daemonize yes
	fi

	(cd "$REPO_ROOT/app" && go run ./cmd/migrate)
	(cd "$REPO_ROOT/app" && go run ./cmd/seed -file "$REPO_ROOT/data/profile.json")

	if ! is_running "$PID_DIR/server.pid"; then
		(
			cd "$REPO_ROOT/app"
			RESUME_DIR="$REPO_ROOT/data" nohup go run ./cmd/server >"$LOG_DIR/server.log" 2>&1 &
			echo $! >"$PID_DIR/server.pid"
		)
		echo "server starting, logs: $LOG_DIR/server.log"
	fi

	if ! is_running "$PID_DIR/frontend.pid"; then
		(
			cd "$REPO_ROOT/frontend"
			VITE_API_BASE_URL="$API_BASE_URL" nohup npx vite --port "$frontend_port" >"$LOG_DIR/frontend.log" 2>&1 &
			echo $! >"$PID_DIR/frontend.pid"
		)
		echo "frontend starting, logs: $LOG_DIR/frontend.log"
	fi

	sleep 2
	echo
	echo "API:      $API_BASE_URL  (curl -u $API_BASIC_AUTH_USER:$API_BASIC_AUTH_PASS $API_BASE_URL/healthz)"
	echo "Frontend: ${FRONTEND_ORIGIN:-http://localhost:$frontend_port}  (login: $API_BASIC_AUTH_USER / $API_BASIC_AUTH_PASS)"
}

case "$action" in
start) start_all ;;
stop) stop_all ;;
status) status_all ;;
*)
	echo "usage: $0 [start|stop|status]" >&2
	exit 1
	;;
esac

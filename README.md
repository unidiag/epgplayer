# epgplayer
Player eit.ts files from epg.by

# Compile
1. Download this repo, enter to dir and run: `make`
2. Move your binary file to bin folder: `cp ./epgplayer-x86 /usr/local/bin/`

# Install ready execute file from EPG.BY
1. Stop and delete your old EIT-player (cherryepg, eit-stream, tsduck, etc..)
2. Copy and move to BIN-dir: `wget -O /usr/local/bin/epgplayer-x86 https://epg.by/epgplayer-x86 && chmod +x /usr/local/bin/epgplayer-x86`

# Run
1. Make your SHELL-file and edit your <token> and <multicast_addr>
```
#!/bin/sh
# EPGPlayer launcher helper
# - update: download latest binary to /usr/local/bin and chmod +x
# - start/stop/restart/status: manage a detached screen session running epgplayer-x86

set -eu

### ---- Config (edit to your needs) ----
BIN="/usr/local/bin/epgplayer-x86"
URL="https://epg.by/epgplayer-x86"
SCREEN_NAME="epgplayer"

# Default runtime params
TOKEN="7jD3hKhIcLpB9DbM"
DST="udp://lo@239.1.1.50:5500"
### -------------------------------------

log() { printf '%s\n' "$*" >&2; }

need_cmd() {
  command -v "$1" >/dev/null 2>&1 || {
    log "Missing dependency: $1"; exit 127;
  }
}

update_bin() {
  need_cmd wget
  log "Updating $BIN from $URL ..."
  wget -O "$BIN" "$URL" && chmod +x "$BIN"
  log "Update done: $BIN"
}

is_running() {
  # Return 0 if process exists, 1 otherwise
  pgrep -x "$(basename "$BIN")" >/dev/null 2>&1
}

stop() {
  # Try to stop by screen session first, then kill leftover process
  if screen -ls | grep -q "[.]$SCREEN_NAME"; then
    log "Stopping screen session $SCREEN_NAME ..."
    screen -S "$SCREEN_NAME" -X quit || true
    sleep 0.2
  fi

  if is_running; then
    PID="$(pgrep -x "$(basename "$BIN")" || true)"
    if [ -n "${PID:-}" ]; then
      log "Killing process $PID ..."
      kill "$PID" 2>/dev/null || true
      # Fallback to -9 if not gone quickly
      for _ in 1 2 3; do
        is_running || break
        sleep 0.2
      done
      is_running && kill -9 "$PID" 2>/dev/null || true
    fi
  fi
}

start() {
  need_cmd screen

  if [ ! -x "$BIN" ]; then
    log "Binary not found or not executable: $BIN"
    log "Run: $0 update"
    exit 1
  fi

  # Allow overriding TOKEN/DST via env or args
  _TOKEN="${1:-$TOKEN}"
  _DST="${2:-$DST}"

  log "Starting in screen session '$SCREEN_NAME' ..."
  screen -dmS "$SCREEN_NAME" "$BIN" "$_TOKEN" "$_DST"
  sleep 0.2
  status
}

status() {
  if is_running; then
    PID="$(pgrep -x "$(basename "$BIN")")"
    log "Running: $(basename "$BIN") (pid $PID)"
    screen -ls | grep -q "[.]$SCREEN_NAME" && log "Screen session: $SCREEN_NAME"
  else
    log "Not running."
  fi
}

usage() {
  cat <<EOF
Usage: $0 <command> [args]

Commands:
  update               Download binary to $BIN and chmod +x
  start [TOKEN] [DST]  Start in detached screen (defaults from config)
  stop                 Stop running process/screen session
  restart              Stop then start
  status               Show running status

Examples:
  $0 update
  $0 start
  $0 start 7jD3hKhIcLpB9DbM udp://lo@239.1.1.50:5500
EOF
}

CMD="${1:-restart}"
case "$CMD" in
  update)
    update_bin
    ;;
  start)
    shift || true
    start "$@"
    ;;
  stop)
    stop
    ;;
  restart)
    stop
    start
    ;;
  status)
    status
    ;;
  *)
    usage
    exit 2
    ;;
esac

```
2. Put this shell script into startup

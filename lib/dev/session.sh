# dev-worktree session commands: code, shell, dash, _watcher, _transcript
# Sourced by bin/dev — do not execute directly.

# ─── Dashboard internals ──────────────────────────────────────

# Render status table in-place (no flicker).
# Uses cursor repositioning instead of screen clear.
_render_status() {
  local session="$1"
  tput cup 0 0 2>/dev/null || true

  echo "dev-worktree dashboard                              "
  echo ""

  local containers
  containers=$(docker ps -a --filter "label=dev-worktree" \
    --format '{{.Label "dev-worktree"}}\t{{.Label "dev-worktree.path"}}\t{{.State}}' 2>/dev/null || true)

  if [ -z "$containers" ]; then
    echo "No environments. Run 'dev up <name>' in another terminal.    "
    # Clear remaining lines
    tput ed 2>/dev/null || true
    return
  fi

  printf "  %-20s %-8s\n" "NAME" "STATUS"
  printf "  %-20s %-8s\n" "----" "------"

  local -A _seen=()
  local key="" wt_path="" state=""
  local status=""
  while IFS=$'\t' read -r key wt_path state; do
    [ -z "$key" ] && continue
    [[ -v _seen["$key"] ]] && continue
    _seen["$key"]=1

    status="stopped"
    echo "$containers" | grep -F "${key}"$'\t' | grep -q "running" && status="running"

    printf "  %-20s %-8s\n" "$key" "$status"
  done <<< "$containers"

  echo ""
  echo "C-c: quit | C-b D: detach"

  # Clear any leftover lines from previous render
  tput ed 2>/dev/null || true

  # ── Detect and add transcript panes for new containers ──
  local existing_titles
  existing_titles=$(tmux list-panes -t "$session" -F '#{pane_title}' 2>/dev/null || true)

  local cid="" dev_cmd=""
  while IFS=$'\t' read -r key wt_path state; do
    [ -z "$key" ] && continue
    [ "$state" != "running" ] && continue
    echo "$existing_titles" | grep -qFx "$key" && continue

    cid=$(docker ps -q --filter "label=dev-worktree=$key" --filter "status=running" 2>/dev/null | head -1)
    [ -z "$cid" ] && continue

    dev_cmd=$(realpath "$0" 2>/dev/null || echo "$0")

    tmux split-window -h -t "$session" \
      -e "DEV_KEY=$key" \
      -e "DEV_CID=$cid" \
      -e "DEV_CMD=$dev_cmd" \
      '"${DEV_CMD}" _transcript "${DEV_KEY}" "${DEV_CID}"'
    tmux select-pane -T "$key"
    # Layout: status sidebar on left (~30%), transcript panes stacked on right
    tmux select-layout -t "$session" main-vertical
    tmux resize-pane -t "${session}:.0" -x 60
  done <<< "$containers"
}

# Internal: tmux dashboard watcher. Called via `dev _watcher` from tmux.
# Uses docker events for real-time updates instead of polling.
_watcher() {
  local session="$1"
  local poll_interval="${DEV_POLL_INTERVAL:-5}"

  # On exit (Ctrl+C, etc.), kill the entire tmux session
  trap 'tmux kill-session -t "'"$session"'" 2>/dev/null; exit 0' EXIT

  tput clear 2>/dev/null || clear

  while true; do
    _render_status "$session" || true
    sleep "$poll_interval"
  done
}

# Internal: stream Claude transcript for a container.
# Called via `dev _transcript <key> <container_id>` from tmux pane.
_transcript() {
  local key="$1" cid="$2"
  local _tail_pid=""

  _kill_tail() { [ -n "$_tail_pid" ] && kill "$_tail_pid" 2>/dev/null; _tail_pid=""; }
  trap '_kill_tail; exit 0' SIGTERM SIGINT EXIT

  echo "[$key] Waiting for Claude transcript..."
  echo ""

  # Find the latest session JSONL file in the container (exclude history.jsonl)
  _find_latest_jsonl() {
    docker exec "$cid" sh -c '
      files=$(find "$HOME/.claude" -name "*.jsonl" ! -name "history.jsonl" -type f 2>/dev/null)
      [ -z "$files" ] && exit 0
      echo "$files" | xargs ls -t 2>/dev/null | head -1
    ' 2>/dev/null || true
  }

  # jq filter for formatting transcript with ANSI colors
  local jq_filter='
    def green:  "\u001b[32m" + . + "\u001b[0m";
    def yellow: "\u001b[33m" + . + "\u001b[0m";
    def blue:   "\u001b[34m" + . + "\u001b[0m";
    def dim:    "\u001b[2m" + . + "\u001b[0m";
    if .type == "user" then
      if (.message.content | type) == "string" then
        ("[user] " + .message.content) | green
      else empty
      end
    elif .type == "assistant" then
      if .message.content then
        [.message.content[] |
          if .type == "text" then ("[asst] " + .text) | yellow
          elif .type == "tool_use" then ("[tool] " + .name + "(" + (.input | tostring | .[0:80]) + ")") | blue
          else empty
          end
        ] | join("\n")
      else empty
      end
    elif .type == "system" and .subtype == "turn_duration" then
      ("[done] " + (.durationMs / 1000 | tostring) + "s") | dim
    else empty
    end
  '

  local current_jsonl=""
  local printed_waiting=false

  while true; do
    # Check if container is still running
    docker inspect --format '{{.State.Running}}' "$cid" 2>/dev/null | grep -q true || {
      _kill_tail
      echo "[$key] Container stopped."
      sleep infinity &
      wait $! 2>/dev/null || exit 0
    }

    local latest_jsonl=""
    latest_jsonl=$(_find_latest_jsonl)

    if [ -z "$latest_jsonl" ]; then
      if [ "$printed_waiting" = false ]; then
        echo "[$key] Waiting for Claude session..."
        printed_waiting=true
      fi
      sleep 5
      continue
    fi

    # New JSONL detected — kill old tail and start new one
    if [ "$latest_jsonl" != "$current_jsonl" ]; then
      _kill_tail
      # On switch, only show new lines (-n 0); first session shows all (-n +1)
      local tail_opt="-n 0"
      [ -z "$current_jsonl" ] && tail_opt="-n +1"
      current_jsonl="$latest_jsonl"
      printed_waiting=false
      echo ""
      echo "[$key] Streaming: $(basename "$current_jsonl")"
      echo ""

      # Run tail|jq in background so we can keep checking for newer files
      (
        trap 'kill 0' SIGTERM
        docker exec "$cid" tail -f $tail_opt "$current_jsonl" 2>/dev/null | \
          docker exec -i "$cid" jq --unbuffered -r "$jq_filter" 2>/dev/null || true
      ) &
      _tail_pid=$!
    fi

    sleep 5
  done
}

# Resolve environment interactively if not specified.
# Sets DEV_SELECTED_KEY (no subshell, so fzf works).
_resolve_name_or_select() {
  DEV_SELECTED_KEY=""
  local running_containers
  running_containers=$(docker ps --filter "label=dev-worktree" --filter "status=running" \
    --format '{{.Label "dev-worktree"}}' 2>/dev/null | sort -u || true)
  local running_count
  running_count=$(echo "$running_containers" | grep -c . 2>/dev/null || echo 0)

  if [ "$running_count" -eq 0 ]; then
    echo "ERROR: No running environments. Run 'dev up <name>' first." >&2
    return 1
  elif [ "$running_count" -eq 1 ]; then
    DEV_SELECTED_KEY=$(echo "$running_containers" | head -1)
  else
    _select_environment "running" || return 1
    # DEV_SELECTED_KEY is set by _select_environment
  fi
}

# ─── dev code ──────────────────────────────────────────────────

cmd_code() {
  local name=""

  while [[ $# -gt 0 ]]; do
    case "$1" in
      -h|--help)
        cat <<EOF
Usage: dev code [name]

Start an AI coding session in a devcontainer.

Without arguments: single env → direct connect, multiple → select.
With a name: directly connects to that environment.

You will be prompted whether to enable --dangerously-skip-permissions.
The base command is determined by WT_EXEC_CMD in .devcontainer/.env
(default: claude).
EOF
        return 0 ;;
      -*) echo "Unknown option: $1" >&2; return 1 ;;
      *)  name="$1"; shift ;;
    esac
  done

  # ── Resolve: interactive select sets DEV_SELECTED_KEY, name arg needs _resolve_key ──
  local dev_key
  if [ -z "$name" ]; then
    _resolve_name_or_select || return 1
    dev_key="$DEV_SELECTED_KEY"
  else
    dev_key=$(_resolve_key "$name")
  fi

  _container_is_running "$dev_key" || \
    { echo "ERROR: Container not running. Run 'dev up ${dev_key#*/}' first." >&2; return 1; }

  local worktree_dir
  worktree_dir=$(_get_wt_path "$dev_key")
  [ -z "$worktree_dir" ] && { echo "ERROR: Cannot resolve worktree path for '$dev_key'." >&2; return 1; }

  local exec_cmd
  exec_cmd=$(_get_exec_cmd "$worktree_dir")

  local -a exec_args
  read -ra exec_args <<< "$exec_cmd"

  # Ask whether to enable --dangerously-skip-permissions (claude only)
  if [[ "${exec_args[0]}" == "claude" ]] && [[ "$exec_cmd" != *"--dangerously-skip-permissions"* ]]; then
    printf "Enable --dangerously-skip-permissions? [y/N]: "
    local answer=""
    read -r answer </dev/tty
    if [[ "$answer" =~ ^[Yy]$ ]]; then
      exec_args+=("--dangerously-skip-permissions")
    fi
  fi

  devcontainer exec --workspace-folder "$worktree_dir" \
    --id-label "dev-worktree=$dev_key" "${exec_args[@]}"
}

# ─── dev shell ─────────────────────────────────────────────────

cmd_shell() {
  local name=""

  while [[ $# -gt 0 ]]; do
    case "$1" in
      -h|--help)
        cat <<EOF
Usage: dev shell [name]

Open a shell in a running devcontainer.

Without arguments: single env → direct connect, multiple → select.
With a name: directly connects to that environment.
EOF
        return 0 ;;
      -*) echo "Unknown option: $1" >&2; return 1 ;;
      *)  name="$1"; shift ;;
    esac
  done

  local dev_key
  if [ -z "$name" ]; then
    _resolve_name_or_select || return 1
    dev_key="$DEV_SELECTED_KEY"
  else
    dev_key=$(_resolve_key "$name")
  fi

  _container_is_running "$dev_key" || \
    { echo "ERROR: Container not running. Run 'dev up ${dev_key#*/}' first." >&2; return 1; }

  local worktree_dir
  worktree_dir=$(_get_wt_path "$dev_key")
  [ -z "$worktree_dir" ] && { echo "ERROR: Cannot resolve worktree path for '$dev_key'." >&2; return 1; }

  devcontainer exec --workspace-folder "$worktree_dir" \
    --id-label "dev-worktree=$dev_key" bash
}

# ─── dev dash ──────────────────────────────────────────────────

cmd_dash() {
  while [[ $# -gt 0 ]]; do
    case "$1" in
      -h|--help)
        cat <<EOF
Usage: dev dash

Open a tmux dashboard monitoring all dev-worktree environments.

Each running container's Claude transcript is streamed in a separate pane.
New containers are automatically detected and added via docker events.

Requires: tmux, docker
EOF
        return 0 ;;
      -*) echo "Unknown option: $1" >&2; return 1 ;;
      *)  echo "Unknown argument: $1" >&2; return 1 ;;
    esac
  done

  if ! command -v tmux &>/dev/null; then
    echo "ERROR: tmux required. Install: brew install tmux" >&2
    return 1
  fi

  local session="dev-worktree"
  local dev_cmd
  dev_cmd=$(realpath "$0" 2>/dev/null || echo "$0")

  tmux kill-session -t "$session" 2>/dev/null || true

  tmux new-session -d -s "$session" -n "dash" \
    -e "DEV_CMD=$dev_cmd" -e "DEV_SESSION=$session" \
    '"${DEV_CMD}" _watcher "${DEV_SESSION}"'
  tmux select-pane -t "$session" -T "_dashboard"

  # Re-apply layout on window resize to keep sidebar width fixed
  tmux set-hook -t "$session" after-resize-window \
    "select-layout main-vertical ; resize-pane -t 0 -x 60"

  # Ctrl+C in any pane kills the dashboard session
  # Binding is cleaned up after detach/exit to avoid affecting other sessions
  tmux bind-key -T root C-c kill-session -t "$session"

  tmux attach-session -t "$session"

  # Clean up global binding after session ends or detaches
  tmux unbind-key -T root C-c 2>/dev/null || true
}

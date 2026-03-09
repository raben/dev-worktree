# dev-worktree session commands: code, shell, dash, _watcher
# Sourced by bin/dev — do not execute directly.

# Internal: tmux dashboard watcher. Called via `dev _watcher` from tmux.
# Not intended for direct user invocation.
_watcher() {
  trap 'exit 0' SIGTERM SIGINT
  local session="$1" filter="${2:-}"
  local poll_interval="${DEV_POLL_INTERVAL:-3}"

  while true; do
    tput clear 2>/dev/null || clear
    echo "dev-worktree dashboard  (poll: ${poll_interval}s)"
    echo ""

    local containers
    containers=$(docker ps -a --filter "label=dev-worktree" \
      --format '{{.Label "dev-worktree"}}\t{{.Label "dev-worktree.path"}}\t{{.State}}' 2>/dev/null || true)

    if [ -z "$containers" ]; then
      echo "No environments. Run 'dev up <name>' in another terminal."
    else
      printf "  %-25s %-10s %s\n" "NAME" "STATUS" "PORTS"
      printf "  %-25s %-10s %s\n" "----" "------" "-----"

      local _seen=""
      local key="" wt_path="" state=""
      while IFS=$'\t' read -r key wt_path state; do
        [ -z "$key" ] && continue
        echo "$_seen" | grep -qxF "$key" && continue
        _seen="${_seen}${key}"$'\n'

        local status="stopped"
        echo "$containers" | grep "^${key}"$'\t' | grep -q "running" && status="running"

        local ports=""
        [ -n "$wt_path" ] && ports=$(_get_ports_from_env "$wt_path")

        printf "  %-25s %-10s %s\n" "$key" "$status" "$ports"
      done <<< "$containers"
    fi

    echo ""
    echo "Ctrl+B then arrow keys to switch panes. Ctrl+B D to detach."

    # ── Detect and add new panes ──
    local existing_titles
    existing_titles=$(tmux list-panes -t "$session" -F '#{pane_title}' 2>/dev/null || true)

    while IFS=$'\t' read -r key wt_path state; do
      [ -z "$key" ] && continue
      [ "$state" != "running" ] && continue
      [ -n "$filter" ] && [[ "$key" != *"$filter"* ]] && continue
      echo "$existing_titles" | grep -qFx "$key" && continue
      [ -z "$wt_path" ] && continue

      local exec_cmd
      exec_cmd=$(_get_exec_cmd "$wt_path")

      # exec_cmd is intentionally unquoted for word-splitting (e.g., "claude --dangerously-skip-permissions").
      # Safety: _validate_exec_cmd allowlist prevents glob/injection chars.
      tmux split-window -t "$session" -e "DEV_KEY=$key" -e "DEV_WT_PATH=$wt_path" -e "DEV_EXEC_CMD=$exec_cmd" \
        'printf "\033]2;${DEV_KEY}\033\\"; devcontainer exec --workspace-folder "${DEV_WT_PATH}" --id-label "dev-worktree=${DEV_KEY}" ${DEV_EXEC_CMD}; echo "--- Session ended (${DEV_KEY}). Press Enter to close. ---"; read'
      tmux select-pane -T "$key"
      tmux select-layout -t "$session" tiled
    done <<< "$containers"

    sleep "$poll_interval"
  done
}

# Resolve environment name interactively if not specified.
# Outputs the bare name (without project prefix).
_resolve_name_or_select() {
  local running_containers
  running_containers=$(docker ps --filter "label=dev-worktree" --filter "status=running" \
    --format '{{.Label "dev-worktree"}}' 2>/dev/null | sort -u || true)
  local running_count
  running_count=$(echo "$running_containers" | grep -c . 2>/dev/null || echo 0)

  if [ "$running_count" -eq 0 ]; then
    echo "ERROR: No running environments. Run 'dev up <name>' first." >&2
    return 1
  elif [ "$running_count" -eq 1 ]; then
    local selected_key
    selected_key=$(echo "$running_containers" | head -1)
    echo "${selected_key#*/}"
  else
    local selected
    selected=$(_select_environment "running") || return 1
    echo "${selected#*/}"
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

  # ── Resolve name if not specified ──
  if [ -z "$name" ]; then
    name=$(_resolve_name_or_select) || return 1
  fi

  local dev_key
  dev_key=$(_resolve_key "$name")

  _container_is_running "$dev_key" || \
    { echo "ERROR: Container not running. Run 'dev up $name' first." >&2; return 1; }

  local worktree_dir
  worktree_dir=$(_get_wt_path "$dev_key")
  [ -z "$worktree_dir" ] && { echo "ERROR: Cannot resolve worktree path for '$dev_key'." >&2; return 1; }

  local exec_cmd
  exec_cmd=$(_get_exec_cmd "$worktree_dir")

  # Ask whether to enable --dangerously-skip-permissions (claude only)
  if [[ "$exec_cmd" == "claude" || "$exec_cmd" == "claude "* ]] && [[ "$exec_cmd" != *"--dangerously-skip-permissions"* ]]; then
    printf "Enable --dangerously-skip-permissions? [y/N]: "
    local answer=""
    read -r answer </dev/tty
    if [[ "$answer" =~ ^[Yy]$ ]]; then
      exec_cmd="$exec_cmd --dangerously-skip-permissions"
      _validate_exec_cmd "$exec_cmd"
    fi
  fi

  # exec_cmd is intentionally unquoted for word-splitting (e.g., "claude --dangerously-skip-permissions").
  # Safety: _validate_exec_cmd allowlist prevents glob/injection chars.
  # --id-label is required because devcontainer up uses custom labels (replaces default identification).
  devcontainer exec --workspace-folder "$worktree_dir" \
    --id-label "dev-worktree=$dev_key" $exec_cmd
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

  # ── Resolve name if not specified ──
  if [ -z "$name" ]; then
    name=$(_resolve_name_or_select) || return 1
  fi

  local dev_key
  dev_key=$(_resolve_key "$name")

  _container_is_running "$dev_key" || \
    { echo "ERROR: Container not running. Run 'dev up $name' first." >&2; return 1; }

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

Open a tmux dashboard showing all environments.

Running environments are automatically detected and opened as tmux panes.
New environments started with 'dev up' are added automatically.

The command in each pane is determined by WT_EXEC_CMD in .devcontainer/.env
(default: claude).

Requires 'tmux'.

Environment variables:
  DEV_POLL_INTERVAL   Dashboard poll interval in seconds (default: 3)
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

  local session="dev"
  local dev_cmd
  dev_cmd=$(realpath "$0" 2>/dev/null || echo "$0")

  tmux kill-session -t "$session" 2>/dev/null || true

  tmux new-session -d -s "$session" -n "dev" \
    -e "DEV_CMD=$dev_cmd" -e "DEV_SESSION=$session" \
    '"${DEV_CMD}" _watcher "${DEV_SESSION}"'
  tmux select-pane -t "$session" -T "_dashboard"

  tmux attach-session -t "$session"
}

# dev-worktree utilities
# Sourced by bin/dev — do not execute directly.

_trim() {
  local s="$1"
  s="${s#"${s%%[![:space:]]*}"}"
  s="${s%"${s##*[![:space:]]}"}"
  printf '%s' "$s"
}

_spinner_start() {
  local msg="${1:-Working}"
  if [ -t 1 ]; then
    printf '%s ' "$msg"
    (while true; do printf '.'; sleep 1; done) &
    _SPINNER_PID=$!
  else
    echo "$msg"
    _SPINNER_PID=""
  fi
}

_spinner_stop() {
  if [ -n "${_SPINNER_PID:-}" ]; then
    kill "$_SPINNER_PID" 2>/dev/null || true
    wait "$_SPINNER_PID" 2>/dev/null || true
    _SPINNER_PID=""
    echo ""
  fi
}

_validate_name() {
  local name="$1"
  if [[ ! "$name" =~ ^[a-zA-Z0-9][a-zA-Z0-9._/-]*$ ]]; then
    echo "ERROR: Invalid name '$name'. Use alphanumeric, dots, hyphens, slashes." >&2
    return 1
  fi
  if [[ "$name" == *".."* ]]; then
    echo "ERROR: Name must not contain '..'." >&2
    return 1
  fi
}

# Allowlist-based validation: only safe chars for shell word-splitting.
_validate_exec_cmd() {
  local cmd="$1"
  if [[ ! "$cmd" =~ ^[a-zA-Z0-9\ _./:=@,+-]+$ ]]; then
    echo "ERROR: WT_EXEC_CMD contains unsafe characters: $cmd" >&2
    echo "Allowed: alphanumeric, spaces, dots, hyphens, underscores, slashes, colons, equals, @, commas" >&2
    return 1
  fi
}

# Select environment interactively. Sets DEV_SELECTED_KEY (no subshell).
# fzf must run outside $() to access /dev/tty reliably.
_select_environment() {
  DEV_SELECTED_KEY=""
  local status_filter="${1:-}"  # "running" to show only running containers
  local containers
  local docker_args=(docker ps --filter "label=dev-worktree" --format '{{.Label "dev-worktree"}}\t{{.State}}')
  [ -z "$status_filter" ] && docker_args=(docker ps -a --filter "label=dev-worktree" --format '{{.Label "dev-worktree"}}\t{{.State}}')
  [ -n "$status_filter" ] && docker_args+=(--filter "status=$status_filter")
  containers=$("${docker_args[@]}" 2>/dev/null || true)

  if [ -z "$containers" ]; then
    echo "ERROR: No dev-worktree environments found. Run 'dev up <name>' first." >&2
    return 1
  fi

  # Deduplicate keys
  local _seen=""
  local keys=()
  while IFS=$'\t' read -r key state; do
    [ -z "$key" ] && continue
    case "$_seen" in *"|$key|"*) continue ;; esac
    _seen="${_seen}|${key}|"
    keys+=("$key ($state)")
  done <<< "$containers"

  if [ "${#keys[@]}" -eq 1 ]; then
    DEV_SELECTED_KEY="${keys[0]% (*}"
    _validate_name "$DEV_SELECTED_KEY" || return 1
    return 0
  fi

  local selected=""
  selected=$(printf '%s\n' "${keys[@]}" | fzf --prompt="Select environment: " --reverse) || {
    echo "ERROR: Selection cancelled." >&2
    return 1
  }
  [ -z "$selected" ] && { echo "ERROR: No selection." >&2; return 1; }
  DEV_SELECTED_KEY="${selected% (*}"
  _validate_name "$DEV_SELECTED_KEY" || return 1
}

# ─── Port Allocation ─────────────────────────────────────────

# Check port availability with lsof (macOS) or ss (Linux) fallback.
_port_is_available() {
  if command -v lsof &>/dev/null; then
    ! lsof -i ":${1}" -sTCP:LISTEN &>/dev/null
  elif command -v ss &>/dev/null; then
    ! ss -tlnH "sport = :${1}" 2>/dev/null | grep -q .
  else
    echo "WARNING: Cannot check port availability (no lsof or ss)" >&2
    return 0
  fi
}

# Read .env.example → allocate available ports → output KEY=VALUE lines.
# Port availability is checked via lsof/ss. Note: TOCTOU race exists between
# check and actual bind by devcontainer; mitigated by the mkdir lock in cmd_up.
_allocate_ports() {
  local env_example="$1"
  local var_name="" base_value="" port=""

  while IFS='=' read -r var_name base_value || [ -n "$var_name" ]; do
    [[ -z "$var_name" || "$var_name" =~ ^[[:space:]]*# ]] && continue
    var_name=$(_trim "$var_name")
    base_value=$(_trim "$base_value")

    if [[ "$var_name" =~ _PORT$ ]]; then
      if ! [[ "$base_value" =~ ^[0-9]+$ ]]; then
        echo "ERROR: Invalid port value for $var_name: $base_value" >&2
        return 1
      fi
      if [ "$base_value" -lt 1024 ] || [ "$base_value" -gt 65535 ]; then
        echo "ERROR: Port $base_value out of valid range (1024-65535) for $var_name" >&2
        return 1
      fi
      port=$base_value
      while ! _port_is_available "$port"; do
        port=$((port + 1))
        if [ "$port" -gt 65535 ]; then
          echo "ERROR: No available port for $var_name (base: $base_value)" >&2
          return 1
        fi
      done
      echo "${var_name}=${port}"
    else
      echo "${var_name}=${base_value}"
    fi
  done < "$env_example"
}

# Read ports from worktree .env file
_get_ports_from_env() {
  local wt_path="$1"
  local env_file="$wt_path/.devcontainer/.env"
  [ -f "$env_file" ] || return 0
  local k="" v=""
  { grep '_PORT=' "$env_file" || true; } | while IFS='=' read -r k v; do
    printf '%s=%s ' "$(echo "$k" | sed 's/^WT_//; s/_PORT$//')" "$v"
  done
}

# Get WT_EXEC_CMD from worktree .env (fallback: claude)
_get_exec_cmd() {
  local wt_path="$1"
  local cmd=""
  local env_file="$wt_path/.devcontainer/.env"
  if [ -f "$env_file" ]; then
    cmd=$(grep '^WT_EXEC_CMD=' "$env_file" 2>/dev/null | head -1 | cut -d= -f2-)
  fi
  if [ -z "$cmd" ]; then
    cmd="claude"
  fi
  _validate_exec_cmd "$cmd" || return 1
  echo "$cmd"
}

# Check if a container with the given dev-worktree label key is running.
_container_is_running() {
  local key="$1"
  local count
  count=$(docker ps --filter "label=dev-worktree=$key" --filter "status=running" -q 2>/dev/null | wc -l | tr -d ' ')
  [ "$count" -gt 0 ]
}

# Resolve key: prepend project name if bare name given.
# Returns error if not in a git repo and key has no '/'.
_resolve_key() {
  local key="$1"
  if [[ "$key" != */* ]]; then
    local project_dir
    project_dir=$(git rev-parse --show-toplevel 2>/dev/null || true)
    if [ -z "$project_dir" ]; then
      echo "ERROR: Not in a git repository. Specify full key (project/name) or cd into the project." >&2
      return 1
    fi
    key="$(basename "$project_dir")/$key"
  fi
  echo "$key"
}

# Get worktree path from docker label
_get_wt_path() {
  local key="$1"
  docker ps -a --filter "label=dev-worktree=$key" \
    --format '{{.Label "dev-worktree.path"}}' 2>/dev/null | head -1
}

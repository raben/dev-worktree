# dev-worktree lifecycle commands: up, down, prune, list
# Sourced by bin/dev — do not execute directly.

# ─── dev up ─────────────────────────────────────────────────────

cmd_up() {
  local project_dir="" name="" base_branch="main"

  while [[ $# -gt 0 ]]; do
    case "$1" in
      --project-dir) project_dir="$2"; shift 2 ;;
      -h|--help)
        cat <<EOF
Usage: dev up [OPTIONS] [name] [base-branch]

Create (or resume) a worktree and start its devcontainer.

Arguments:
  name          Worktree / branch name (default: current branch)
  base-branch   Base branch (default: main)

Options:
  --project-dir DIR   Project root (default: git root of CWD)
EOF
        return 0 ;;
      -*) echo "Unknown option: $1" >&2; return 1 ;;
      *)  if [ -z "$name" ]; then name="$1"; else base_branch="$1"; fi; shift ;;
    esac
  done

  if [ -z "$name" ]; then
    name=$(git rev-parse --abbrev-ref HEAD 2>/dev/null || true)
    [ -z "$name" ] || [ "$name" = "HEAD" ] && name="dev-$(od -An -tx1 -N4 /dev/urandom | tr -d ' \n')"
  fi

  # Sanitize: replace path separators and dots with hyphens
  local original_name="$name"
  name=$(printf '%s' "$name" | tr '/' '-')
  if [ "$name" != "$original_name" ]; then
    echo "Sanitized '$original_name' → '$name'"
  fi
  _validate_name "$name"

  if [ -z "$project_dir" ]; then
    project_dir=$(git rev-parse --show-toplevel 2>/dev/null || true)
    [ -z "$project_dir" ] && { echo "ERROR: Not in a git repo. Use --project-dir." >&2; return 1; }
  fi

  local project_name
  project_name=$(basename "$project_dir")
  local dev_key="${project_name}/${name}"
  local branch_name="dev/${project_name}/${name}"
  local worktree_dir
  worktree_dir="$(dirname "$project_dir")/dev/${project_name}/${name}"

  # Validate worktree_dir is within expected parent (defense against path traversal)
  local expected_parent
  expected_parent=$(cd "$(dirname "$project_dir")" && pwd)
  local resolved_wt_dir="${expected_parent}/dev/${project_name}/${name}"
  case "$resolved_wt_dir" in
    "$expected_parent"/*) ;; # ok
    *) echo "ERROR: Worktree path escapes expected parent directory." >&2; return 1 ;;
  esac

  if ! command -v devcontainer &>/dev/null; then
    echo "ERROR: devcontainer CLI not found. Install: npm install -g @devcontainers/cli" >&2
    return 1
  fi

  # ── Protected branches: create new branch from it instead of checking out directly ──
  local protected_branches="main master develop"
  if [[ " $protected_branches " == *" $name "* ]] && [ ! -d "$worktree_dir" ]; then
    base_branch="$name"
    echo "Protected branch '$base_branch'. Creating '${branch_name}' from '$base_branch'."
  fi

  # ── Existing worktree: resume ──
  if [ -d "$worktree_dir" ]; then
    echo "Worktree '${branch_name}' already exists at $worktree_dir"

    # Remove exited containers before resuming (prevents stale state)
    local exited_ids
    exited_ids=$(docker ps -a --filter "label=dev-worktree=$dev_key" --filter "status=exited" -q 2>/dev/null || true)
    if [ -n "$exited_ids" ]; then
      echo "$exited_ids" | while read -r cid; do docker rm -f "$cid" 2>/dev/null || true; done
    fi

    if _container_is_running "$dev_key"; then
      echo "Container already running."
    else
      local _dc_log
      _dc_log=$(mktemp /tmp/dev-worktree-dc-XXXXXX)
      trap '_spinner_stop' EXIT
      _spinner_start "Starting devcontainer"
      if devcontainer up --workspace-folder "$worktree_dir" \
        --id-label "dev-worktree=$dev_key" \
        --id-label "dev-worktree.path=$worktree_dir" \
        --id-label "dev-worktree.version=$VERSION" \
        > "$_dc_log" 2>&1; then
        _spinner_stop
        trap - EXIT
        rm -f "$_dc_log"
      else
        _spinner_stop
        trap - EXIT
        echo "ERROR: devcontainer up failed. See log: $_dc_log" >&2
        return 1
      fi
    fi

    echo ""
    echo "Ready. Use 'dev code' or 'dev shell' to connect."
    return
  fi

  # ── New worktree: create ──
  local lockdir="/tmp/dev-worktree-${dev_key//\//_}.lock"
  if ! mkdir "$lockdir" 2>/dev/null; then
    echo "ERROR: Another 'dev up' for '$dev_key' is already running." >&2
    return 1
  fi

  local _cleanup_enabled=true
  local _branch_created=false

  cleanup() {
    local ec=$?
    _spinner_stop
    rmdir "$lockdir" 2>/dev/null || true
    if [ $ec -ne 0 ] && [ "$_cleanup_enabled" = true ]; then
      echo "ERROR: Failed. Cleaning up..."
      local cid
      for cid in $(docker ps -a --filter "label=dev-worktree=$dev_key" -q 2>/dev/null || true); do
        docker rm -f "$cid" 2>/dev/null || true
      done
      if [ -d "$worktree_dir" ]; then
        git -C "$project_dir" worktree remove --force "$worktree_dir" 2>/dev/null || true
      fi
      if [ "$_branch_created" = true ]; then
        git -C "$project_dir" branch -D "$branch_name" 2>/dev/null || true
      fi
      # Clean up temp files
      rm -f "${worktree_dir}/.devcontainer/docker-compose.yml.tmp" 2>/dev/null || true
      echo "If cleanup was incomplete, run: dev prune $name --force" >&2
    fi
  }
  trap cleanup EXIT

  [ ! -d "$project_dir/.devcontainer" ] && { echo "ERROR: No .devcontainer/ in $project_dir" >&2; return 1; }

  # Allocate ports
  local env_example="$project_dir/.devcontainer/.env.example"
  local allocated_vars=""
  if [ -f "$env_example" ]; then
    echo "Allocating ports..."
    allocated_vars=$(_allocate_ports "$env_example")
  fi

  # Create worktree
  echo "Creating worktree..."
  if git -C "$project_dir" show-ref --verify --quiet "refs/heads/$branch_name" 2>/dev/null; then
    git -C "$project_dir" worktree add "$worktree_dir" "$branch_name"
  else
    git -C "$project_dir" worktree add "$worktree_dir" -b "$branch_name" -- "$base_branch"
    _branch_created=true
  fi

  # Copy .devcontainer from project root to worktree.
  # This overwrites the git-checked-out version with the canonical template,
  # ensuring port substitution and compose configuration work correctly.
  echo "Setting up devcontainer..."
  cp -r "$project_dir/.devcontainer" "$worktree_dir/.devcontainer"

  # Exclude devcontainer CLI artifacts from git status in this worktree
  local _git_dir
  _git_dir=$(git -C "$worktree_dir" rev-parse --git-dir 2>/dev/null)
  if [ -n "$_git_dir" ]; then
    mkdir -p "$_git_dir/info"
    if ! grep -qxF '.devcontainer/.devcontainer/' "$_git_dir/info/exclude" 2>/dev/null; then
      echo '.devcontainer/.devcontainer/' >> "$_git_dir/info/exclude"
    fi
  fi

  # Validate container_name uses WT_NAME (prevent name conflicts across worktrees)
  # Check BEFORE variable substitution, using the original project source
  local compose_file_check="$project_dir/.devcontainer/docker-compose.yml"
  if [ -f "$compose_file_check" ]; then
    local hardcoded_names=""
    hardcoded_names=$(grep 'container_name:' "$compose_file_check" 2>/dev/null | grep -v 'WT_NAME' || true)
    if [ -n "$hardcoded_names" ]; then
      echo "WARNING: docker-compose.yml has hardcoded container_name (may conflict with other worktrees):" >&2
      echo "$hardcoded_names" | sed 's/^/  /' >&2
      echo "  Tip: Use \${WT_NAME:-default} pattern. Run 'dev init' to regenerate." >&2
    fi
  fi

  if [ -n "$allocated_vars" ]; then
    allocated_vars=$(printf '%s\nWT_NAME=%s\nCOMPOSE_PROJECT_NAME=%s' \
      "$allocated_vars" "${project_name}-${name}" "${project_name}-${name}")

    install -m 600 /dev/null "$worktree_dir/.devcontainer/.env"
    printf '%s\n' "$allocated_vars" > "$worktree_dir/.devcontainer/.env"

    # docker-compose reads .env from the same directory automatically,
    # so no need to sed-replace variables in docker-compose.yml.
    # COMPOSE_PROJECT_NAME in .env ensures unique container names.
  fi

  # Start devcontainer
  local _dc_exit=0
  local _dc_log
  _dc_log=$(mktemp /tmp/dev-worktree-dc-XXXXXX)
  _spinner_start "Starting devcontainer"
  devcontainer up --workspace-folder "$worktree_dir" \
    --id-label "dev-worktree=$dev_key" \
    --id-label "dev-worktree.path=$worktree_dir" \
    --id-label "dev-worktree.version=$VERSION" \
    > "$_dc_log" 2>&1 \
    || _dc_exit=$?
  _spinner_stop

  if [ "$_dc_exit" -ne 0 ]; then
    echo "ERROR: devcontainer up failed. See log: $_dc_log" >&2
    # Check if container is running despite devcontainer up failure (e.g. postCreateCommand failed)
    if _container_is_running "$dev_key"; then
      echo "WARNING: Container is running despite errors (postCreateCommand may have failed)." >&2
      echo "  You can connect with 'dev shell' and fix manually." >&2
    else
      # Container not running — let cleanup handle it
      return "$_dc_exit"
    fi
  else
    rm -f "$_dc_log"
  fi

  # Fix Docker volume ownership (volumes are initialized as root:root,
  # but remoteUser needs write access, e.g. ~/.claude for Claude Code)
  local _cid
  _cid=$(docker ps -q --filter "label=dev-worktree=$dev_key" 2>/dev/null | head -1)
  if [ -n "$_cid" ]; then
    docker exec -u root "$_cid" sh -c '
      for d in /home/*/.claude; do
        [ -d "$d" ] || continue
        parent_owner=$(stat -c %u:%g "$(dirname "$d")")
        cur_owner=$(stat -c %u:%g "$d")
        [ "$cur_owner" != "$parent_owner" ] && chown "$parent_owner" "$d"
      done
    ' 2>/dev/null || true
  fi

  # Disable cleanup — worktree created successfully
  _cleanup_enabled=false
  trap - EXIT
  rmdir "$lockdir" 2>/dev/null || true

  # Summary
  echo ""
  echo "=== Worktree created ==="
  echo "  Path:   $worktree_dir"
  echo "  Branch: $branch_name"
  if [ -n "$allocated_vars" ]; then
    echo "  Ports:"
    echo "$allocated_vars" | { grep '_PORT=' || true; } | while IFS='=' read -r k v; do echo "    $k: $v"; done
  fi
  echo ""
  echo "Ready. Use 'dev code' or 'dev shell' to connect."
}

# ─── dev down ───────────────────────────────────────────────────

cmd_down() {
  local dev_key=""

  while [[ $# -gt 0 ]]; do
    case "$1" in
      -h|--help)
        cat <<EOF
Usage: dev down [name]

Stop devcontainer (worktree is kept).
Name omitted: select from running environments.
EOF
        return 0 ;;
      -*) echo "Unknown option: $1" >&2; return 1 ;;
      *)  dev_key="$1"; shift ;;
    esac
  done

  if [ -z "$dev_key" ]; then
    _select_environment || return 1
    dev_key="$DEV_SELECTED_KEY"
  fi
  dev_key=$(_resolve_key "$dev_key")

  echo "Stopping '$dev_key'..."

  local container_ids
  container_ids=$(docker ps --filter "label=dev-worktree=$dev_key" -q 2>/dev/null || true)
  if [ -n "$container_ids" ]; then
    local cid
    echo "$container_ids" | while read -r cid; do docker stop "$cid" 2>/dev/null || true; done
  else
    echo "No running containers found for '$dev_key'." >&2
  fi

  echo "Done."
}

# ─── dev prune ─────────────────────────────────────────────────

cmd_prune() {
  local dev_key=""

  while [[ $# -gt 0 ]]; do
    case "$1" in
      -h|--help)
        cat <<EOF
Usage: dev prune [name]

Remove worktree, containers, and branch.
Prompts for confirmation before deleting.
EOF
        return 0 ;;
      -*) echo "Unknown option: $1" >&2; return 1 ;;
      *)  dev_key="$1"; shift ;;
    esac
  done

  if [ -z "$dev_key" ]; then
    _select_environment || return 1
    dev_key="$DEV_SELECTED_KEY"
  fi
  dev_key=$(_resolve_key "$dev_key")

  local worktree_path
  worktree_path=$(_get_wt_path "$dev_key")
  local name="${dev_key#*/}"
  local project_name="${dev_key%%/*}"
  local branch_name="dev/${dev_key}"

  # Derive project_dir from worktree_path (more reliable than CWD)
  local project_dir=""
  if [ -n "$worktree_path" ] && [ -d "$worktree_path" ]; then
    project_dir=$(git -C "$worktree_path" rev-parse --show-toplevel 2>/dev/null || true)
    # worktree's toplevel is itself; get the main repo via commondir
    local commondir
    commondir=$(git -C "$worktree_path" rev-parse --git-common-dir 2>/dev/null || true)
    if [ -n "$commondir" ]; then
      project_dir=$(cd "$worktree_path" && cd "$commondir" && cd .. && pwd)
    fi
  fi
  # Fallback to CWD-based detection
  if [ -z "$project_dir" ]; then
    project_dir=$(git rev-parse --show-toplevel 2>/dev/null || true)
  fi

  # Fallback: if container was removed manually, infer worktree_path from project_dir
  if [ -z "$worktree_path" ] && [ -n "$project_dir" ]; then
    local inferred="$(dirname "$project_dir")/dev/${project_name}/${name}"
    if [ -d "$inferred" ]; then
      worktree_path="$inferred"
    fi
  fi

  # Confirmation (1st)
  local confirm=""
  read -rp "Remove '$dev_key'? (y/N) " confirm
  [[ "$confirm" != [yY]* ]] && { echo "Aborted."; return 1; }

  # Confirmation (2nd) — uncommitted changes
  if [ -n "$worktree_path" ] && [ -d "$worktree_path" ]; then
    if ! git -C "$worktree_path" diff --quiet 2>/dev/null || \
       ! git -C "$worktree_path" diff --cached --quiet 2>/dev/null; then
      echo "WARNING: Uncommitted changes in $worktree_path"
      git -C "$worktree_path" diff --stat 2>/dev/null
      git -C "$worktree_path" diff --cached --stat 2>/dev/null
      echo "  These changes will be permanently lost."
      read -rp "Are you sure you want to delete? (y/N) " confirm
      [[ "$confirm" != [yY]* ]] && { echo "Aborted."; return 1; }
    fi
  fi

  echo "Removing '$dev_key'..."

  # Stop and remove containers + volumes
  echo "Stopping containers..."

  # Try compose first for proper volumes cleanup
  if [ -n "$worktree_path" ]; then
    local compose_file="$worktree_path/.devcontainer/docker-compose.yml"
    if [ -f "$compose_file" ]; then
      docker compose -f "$compose_file" down -v 2>/dev/null || true
    fi
  fi

  # Fallback: force-remove any remaining containers by label
  local container_ids
  container_ids=$(docker ps -a --filter "label=dev-worktree=$dev_key" -q 2>/dev/null || true)
  if [ -n "$container_ids" ]; then
    local cid
    echo "$container_ids" | while read -r cid; do docker rm -f "$cid" 2>/dev/null || true; done
  fi

  # Remove worktree
  if [ -n "$worktree_path" ] && [ -d "$worktree_path" ]; then
    echo "Removing worktree..."
    if [ -n "$project_dir" ]; then
      # Always use --force: if we reach here, the user already confirmed
      git -C "$project_dir" worktree remove --force "$worktree_path" 2>/dev/null || true
    fi
    # Fallback: if git worktree remove failed, remove directory directly
    if [ -d "$worktree_path" ]; then
      rm -rf "$worktree_path"
    fi
  fi
  if [ -n "$project_dir" ]; then
    git -C "$project_dir" worktree prune 2>/dev/null || true
  fi

  # Clean up empty parent directories (dev/project_name/, dev/)
  if [ -n "$worktree_path" ]; then
    rmdir "$(dirname "$worktree_path")" 2>/dev/null || true
    rmdir "$(dirname "$(dirname "$worktree_path")")" 2>/dev/null || true
  fi

  # Delete branch if merged
  if [ -n "$name" ] && [ -n "$project_dir" ]; then
    if git -C "$project_dir" branch -d "$branch_name" 2>/dev/null; then
      echo "Branch '$branch_name' deleted (merged)."
    else
      echo "Branch '$branch_name' kept (not merged). Use: git branch -D $branch_name"
    fi
  fi

  echo "Done."
}

# ─── dev list ──────────────────────────────────────────────────

cmd_list() {
  local project_dir="" json_output=false

  while [[ $# -gt 0 ]]; do
    case "$1" in
      --project-dir) project_dir="$2"; shift 2 ;;
      --json)        json_output=true; shift ;;
      -h|--help)
        cat <<EOF
Usage: dev list [OPTIONS]

List dev-worktree environments.

Options:
  --project-dir DIR   Filter by project
  --json              Output JSON (requires jq)
EOF
        return 0 ;;
      *) echo "Unknown option: $1" >&2; return 1 ;;
    esac
  done

  local containers
  containers=$(docker ps -a --filter "label=dev-worktree" \
    --format '{{.Label "dev-worktree"}}\t{{.Label "dev-worktree.path"}}\t{{.State}}\t{{.Names}}' 2>/dev/null || true)

  if [ -z "$containers" ]; then
    echo "No dev-worktree environments found."
    return 0
  fi

  if [ -n "$project_dir" ]; then
    local pn
    pn=$(basename "$project_dir")
    local escaped_pn
    escaped_pn=$(printf '%s' "$pn" | sed 's/[.[\*^$()+?{|\\]/\\&/g')
    containers=$(echo "$containers" | { grep "^${escaped_pn}/" || true; })
    if [ -z "$containers" ]; then
      echo "No dev-worktree environments found for $pn."
      return 0
    fi
  fi

  if [ "$json_output" = true ]; then
    if ! command -v jq &>/dev/null; then
      echo "ERROR: jq is required for --json output. Install: brew install jq" >&2
      return 1
    fi
    # Build JSON in a single jq invocation for efficiency
    echo "$containers" | jq -R -s '
      [split("\n")[] | select(length > 0) | split("\t") |
       {key: .[0], path: .[1], state: .[2], container: .[3]}]'
    return 0
  fi

  printf "%-35s %-10s %s\n" "ID" "STATUS" "PORTS"
  printf "%-35s %-10s %s\n" "--" "------" "-----"

  local -A _seen=()
  local key="" wt_path="" state="" cname=""
  local status="" ports=""
  while IFS=$'\t' read -r key wt_path state cname; do
    [ -z "$key" ] && continue
    [[ -v _seen["$key"] ]] && continue
    _seen["$key"]=1

    status="stopped"
    echo "$containers" | grep -F "${key}"$'\t' | grep -q "running" && status="running"

    ports=""
    [ -n "$wt_path" ] && ports=$(_get_ports_from_env "$wt_path")

    printf "%-35s %-10s %s\n" "$key" "$status" "$ports"
  done <<< "$containers"
}

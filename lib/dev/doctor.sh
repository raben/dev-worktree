# dev-worktree doctor command
# Sourced by bin/dev — do not execute directly.

cmd_doctor() {
  while [[ $# -gt 0 ]]; do
    case "$1" in
      -h|--help)
        cat <<EOF
Usage: dev doctor

Check environment health for dev-worktree.
Verifies system dependencies, Docker, Claude credentials,
project configuration, and active environments.
EOF
        return 0 ;;
      *) echo "Unknown option: $1" >&2; return 1 ;;
    esac
  done

  local _fail_count=0 _warn_count=0

  # ── Output helpers ──
  local _green='\033[32m' _red='\033[31m' _yellow='\033[33m' _blue='\033[34m' _bold='\033[1m' _reset='\033[0m'
  if [ ! -t 1 ]; then
    _green='' _red='' _yellow='' _blue='' _bold='' _reset=''
  fi

  _doc_pass()    { printf "  ${_green}✓${_reset} %s\n" "$*"; }
  _doc_fail()    { printf "  ${_red}✗${_reset} %s\n" "$*"; _fail_count=$((_fail_count + 1)); }
  _doc_warn()    { printf "  ${_yellow}⚠${_reset} %s\n" "$*"; _warn_count=$((_warn_count + 1)); }
  _doc_info()    { printf "  ${_blue}ⓘ${_reset} %s\n" "$*"; }
  _doc_section() { printf "\n${_bold}%s${_reset}\n" "$*"; }

  echo "dev doctor - Environment health check"

  # ══════════════════════════════════════════════════════════════
  # 1. System Dependencies
  # ══════════════════════════════════════════════════════════════
  _doc_section "System Dependencies"

  local _deps="docker:Install Docker Desktop
git:brew install git
devcontainer:npm install -g @devcontainers/cli
jq:brew install jq
tmux:brew install tmux
claude:npm install -g @anthropic-ai/claude-code"

  local _dep_name _dep_fix
  while IFS=: read -r _dep_name _dep_fix; do
    [ -z "$_dep_name" ] && continue
    if command -v "$_dep_name" &>/dev/null; then
      local _dep_ver=""
      _dep_ver=$("$_dep_name" --version 2>/dev/null | head -1 | grep -oE '[0-9]+\.[0-9]+[.0-9]*' | head -1 || true)
      if [ -n "$_dep_ver" ]; then
        _doc_pass "$_dep_name ($_dep_ver)"
      else
        _doc_pass "$_dep_name"
      fi
    else
      _doc_fail "$_dep_name — not found. Install: $_dep_fix"
    fi
  done <<< "$_deps"

  # ══════════════════════════════════════════════════════════════
  # 2. Docker Health
  # ══════════════════════════════════════════════════════════════
  _doc_section "Docker"

  if command -v docker &>/dev/null; then
    if docker info &>/dev/null; then
      _doc_pass "Daemon running"
    else
      _doc_fail "Daemon not running — Start Docker Desktop"
    fi

    local _disk_info=""
    _disk_info=$(docker system df --format '{{.Type}}\t{{.Size}}' 2>/dev/null || true)
    if [ -n "$_disk_info" ]; then
      local _disk_parts=""
      while IFS=$'\t' read -r _dtype _dsize; do
        [ -z "$_dtype" ] && continue
        _disk_parts="${_disk_parts}${_disk_parts:+, }${_dtype} ${_dsize}"
      done <<< "$_disk_info"
      _doc_info "Disk: $_disk_parts"
    fi
  else
    _doc_fail "docker not installed — skipping Docker checks"
  fi

  # ══════════════════════════════════════════════════════════════
  # 3. Claude Credentials
  # ══════════════════════════════════════════════════════════════
  _doc_section "Claude Credentials"

  local _claude_dir="$HOME/.claude"

  if [ -d "$_claude_dir" ]; then
    _doc_pass "~/.claude exists"
  else
    _doc_fail "~/.claude not found — Run 'claude' once to log in"
  fi

  # Check for auth files (.claude.json contains auth info including userID)
  if [ -d "$_claude_dir" ]; then
    if [ -f "$_claude_dir/.claude.json" ] && grep -q 'userID' "$_claude_dir/.claude.json" 2>/dev/null; then
      _doc_pass "Auth credentials found"
    elif [ -f "$_claude_dir/credentials.json" ] || [ -f "$_claude_dir/.credentials.json" ]; then
      _doc_pass "Auth credentials found"
    else
      _doc_fail "No auth credentials found — Run 'claude' to log in"
    fi
  fi

  # Check mount in devcontainer.json (project-level)
  local _project_dir=""
  _project_dir=$(git rev-parse --show-toplevel 2>/dev/null || true)

  if [ -n "$_project_dir" ]; then
    local _dc_json="$_project_dir/.devcontainer/devcontainer.json"
    if [ -f "$_dc_json" ]; then
      if grep -q '\.claude' "$_dc_json" 2>/dev/null; then
        _doc_pass "Mount configured in devcontainer.json"
      else
        _doc_fail "No .claude mount in devcontainer.json — Run 'dev init' to regenerate"
      fi

      if grep -q 'CLAUDE_CONFIG_DIR' "$_dc_json" 2>/dev/null; then
        _doc_pass "CLAUDE_CONFIG_DIR set"
      else
        _doc_fail "CLAUDE_CONFIG_DIR not set in devcontainer.json — Run 'dev init' to regenerate"
      fi
    fi
  fi

  # ══════════════════════════════════════════════════════════════
  # 4. Project Health (git repo only)
  # ══════════════════════════════════════════════════════════════
  if [ -n "$_project_dir" ]; then
    local _project_name
    _project_name=$(basename "$_project_dir")
    _doc_section "Project: $_project_name"

    if [ -d "$_project_dir/.devcontainer" ]; then
      _doc_pass ".devcontainer/ exists"
    else
      _doc_fail ".devcontainer/ not found — Run 'dev init'"
    fi

    local _dc_files="Dockerfile
devcontainer.json
docker-compose.yml"
    local _dcf
    while IFS= read -r _dcf; do
      [ -z "$_dcf" ] && continue
      if [ -f "$_project_dir/.devcontainer/$_dcf" ]; then
        if [ "$_dcf" = "devcontainer.json" ] && command -v jq &>/dev/null; then
          if jq empty "$_project_dir/.devcontainer/$_dcf" 2>/dev/null; then
            _doc_pass "$_dcf (valid)"
          else
            _doc_fail "$_dcf (invalid JSON) — Run 'dev init' to regenerate"
          fi
        else
          _doc_pass "$_dcf"
        fi
      else
        _doc_fail "$_dcf not found — Run 'dev init' to regenerate"
      fi
    done <<< "$_dc_files"

    if [ -f "$_project_dir/.devcontainer/.env.example" ]; then
      _doc_pass ".env.example"
    else
      _doc_fail ".env.example not found — Run 'dev init' to regenerate"
    fi

    # Check container_name uses WT_NAME pattern
    local _compose="$_project_dir/.devcontainer/docker-compose.yml"
    if [ -f "$_compose" ]; then
      local _hardcoded=""
      _hardcoded=$(grep 'container_name:' "$_compose" 2>/dev/null | grep -v 'WT_NAME' || true)
      if [ -n "$_hardcoded" ]; then
        _doc_fail "container_name is hardcoded (will conflict across worktrees) — Use \${WT_NAME:-name} pattern"
      else
        _doc_pass "container_name uses \${WT_NAME} pattern"
      fi
    fi
  fi

  # ══════════════════════════════════════════════════════════════
  # 5. Active Environments
  # ══════════════════════════════════════════════════════════════
  if command -v docker &>/dev/null && docker info &>/dev/null; then
    local _containers=""
    _containers=$(docker ps -a --filter "label=dev-worktree" \
      --format '{{.ID}}\t{{.Label "dev-worktree"}}\t{{.State}}' 2>/dev/null || true)

    if [ -n "$_containers" ]; then
      _doc_section "Active Environments"

      local _seen_env="" _cid _ekey _estate
      while IFS=$'\t' read -r _cid _ekey _estate; do
        [ -z "$_ekey" ] && continue
        case "$_seen_env" in *"|$_ekey|"*) continue ;; esac
        _seen_env="${_seen_env}|${_ekey}|"

        if [ "$_estate" = "running" ]; then
          _doc_pass "$_ekey (running)"

          # Check Claude mount inside container
          local _container_home=""
          _container_home=$(docker exec "$_cid" sh -c 'echo $HOME' 2>/dev/null || true)

          if [ -n "$_container_home" ]; then
            if docker exec "$_cid" test -d "${_container_home}/.claude" 2>/dev/null; then
              _doc_pass "  Claude mounted at ${_container_home}/.claude"
            else
              _doc_fail "  Claude not mounted in container — Check devcontainer.json mounts"
            fi

            # Check auth files inside container
            if docker exec "$_cid" sh -c "grep -q userID '${_container_home}/.claude/.claude.json' 2>/dev/null || test -f '${_container_home}/.claude/credentials.json'" 2>/dev/null; then
              _doc_pass "  Claude auth credentials found"
            else
              _doc_fail "  Claude auth credentials not found in container"
            fi
          fi

          # Check claude CLI inside container
          local _container_claude_ver=""
          _container_claude_ver=$(docker exec "$_cid" claude --version 2>/dev/null | head -1 | grep -oE '[0-9]+\.[0-9]+[.0-9]*' | head -1 || true)
          if [ -n "$_container_claude_ver" ]; then
            _doc_pass "  claude CLI available ($_container_claude_ver)"
          else
            if docker exec "$_cid" sh -c 'command -v claude' &>/dev/null 2>/dev/null; then
              _doc_pass "  claude CLI available"
            else
              _doc_fail "  claude CLI not found in container"
            fi
          fi
        else
          _doc_warn "$_ekey ($_estate) — Run: dev prune ${_ekey#*/}"
        fi
      done <<< "$_containers"
    fi
  fi

  # ══════════════════════════════════════════════════════════════
  # Summary
  # ══════════════════════════════════════════════════════════════
  echo ""
  local _total=$((_fail_count + _warn_count))
  if [ "$_total" -eq 0 ]; then
    printf "${_green}All checks passed.${_reset}\n"
  else
    local _parts=""
    [ "$_fail_count" -gt 0 ] && _parts="${_fail_count} error(s)"
    [ "$_warn_count" -gt 0 ] && _parts="${_parts}${_parts:+, }${_warn_count} warning(s)"
    printf "${_yellow}%s found. See above for details.${_reset}\n" "$_parts"
  fi

  return "$_fail_count"
}

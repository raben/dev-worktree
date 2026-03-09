# dev-worktree issue command
# Sourced by bin/dev — do not execute directly.

cmd_issue() {
  local issue_nums=()
  local dry_run=false

  while [[ $# -gt 0 ]]; do
    case "$1" in
      -h|--help)
        cat <<EOF
Usage: dev issue [OPTIONS] [number,...]

Assign GitHub Issues to autonomous AI agents.

Each issue gets its own worktree + devcontainer. The agent works
autonomously and creates a PR when done.

Arguments:
  number,...   Comma-separated issue numbers (e.g. 123 or 12,34,56)
               Without arguments: interactively create a new issue first.

Options:
  --dry-run    Show what would happen without executing

Monitor progress with 'dev dash' or 'tmux attach -t dev-issue-N'.

Requires: gh, tmux
EOF
        return 0 ;;
      --dry-run) dry_run=true; shift ;;
      -*) echo "Unknown option: $1" >&2; return 1 ;;
      *)
        # Split comma-separated numbers
        IFS=',' read -ra _nums <<< "$1"
        for _n in "${_nums[@]}"; do
          _n="${_n#\#}"  # strip leading #
          _n=$(echo "$_n" | tr -d ' ')
          [[ "$_n" =~ ^[0-9]+$ ]] || { echo "ERROR: Invalid issue number: $_n" >&2; return 1; }
          issue_nums+=("$_n")
        done
        shift ;;
    esac
  done

  # ── Dependency check ──
  for _dep in gh tmux jq; do
    command -v "$_dep" &>/dev/null || { echo "ERROR: '$_dep' is required but not found." >&2; return 1; }
  done

  $dry_run && echo "[DRY RUN] No changes will be made."

  # ── Interactive mode: create issue first ──
  if [ "${#issue_nums[@]}" -eq 0 ]; then
    echo "No issue number specified. Creating a new issue..."
    echo ""
    printf "Title: "
    local issue_title=""
    read -r issue_title </dev/tty
    [ -z "$issue_title" ] && { echo "ERROR: Title is required." >&2; return 1; }

    printf "Body (single line, or empty): "
    local issue_body=""
    read -r issue_body </dev/tty

    if $dry_run; then
      echo "[DRY RUN] Would create issue: $issue_title"
      echo "[DRY RUN] Using dummy issue number: 9999"
      issue_nums+=("9999")
    else
      local create_args=(gh issue create --title "$issue_title")
      [ -n "$issue_body" ] && create_args+=(--body "$issue_body")

      local created_url
      created_url=$("${create_args[@]}" 2>&1) || { echo "ERROR: Failed to create issue: $created_url" >&2; return 1; }
      local created_num
      created_num=$(echo "$created_url" | grep -oE '[0-9]+$')
      echo "Created issue #$created_num: $created_url"
      echo ""
      issue_nums+=("$created_num")
    fi
  fi

  # ── Process each issue ──
  local dev_cmd
  dev_cmd=$(realpath "$0" 2>/dev/null || echo "$0")

  for issue_num in "${issue_nums[@]}"; do
    echo "── Issue #$issue_num ──"

    # 1. Fetch issue
    local issue_title="" issue_body="" issue_labels=""
    if $dry_run; then
      issue_title="[DRY RUN] Sample issue title"
      issue_body="This is a dry run. No actual issue fetched."
      issue_labels=""
      echo "  Title: $issue_title"
    else
      local issue_json
      issue_json=$(gh issue view "$issue_num" --json number,title,body,labels 2>&1) || \
        { echo "ERROR: Failed to fetch issue #$issue_num: $issue_json" >&2; continue; }

      issue_title=$(echo "$issue_json" | jq -r '.title')
      issue_body=$(echo "$issue_json" | jq -r '.body // ""')
      issue_labels=$(echo "$issue_json" | jq -r '[.labels[].name] | join(", ")')
      echo "  Title: $issue_title"
    fi

    # 2. Create environment
    if $dry_run; then
      echo "  [DRY RUN] Would run: dev up issue-$issue_num"
    else
      "$dev_cmd" up "issue-$issue_num" 2>&1 | sed 's/^/  /'
    fi

    local dev_key
    dev_key=$(_resolve_key "issue-$issue_num")

    if $dry_run; then
      local wt_path="/tmp/dev/<project>/issue-$issue_num"
      echo "  [DRY RUN] Would write .dev-issue.md to $wt_path"
    else
      _container_is_running "$dev_key" || \
        { echo "ERROR: Container for issue-$issue_num is not running." >&2; continue; }

      local wt_path
      wt_path=$(_get_wt_path "$dev_key")
      [ -z "$wt_path" ] && { echo "ERROR: Cannot resolve worktree path for issue-$issue_num." >&2; continue; }

      # 3. Write issue file to worktree
      cat > "$wt_path/.dev-issue.md" << ISSUEEOF
# Issue #$issue_num: $issue_title

${issue_labels:+Labels: $issue_labels}

$issue_body
ISSUEEOF
    fi

    # 4. Launch agent in tmux (background)
    local session_name="dev-issue-$issue_num"
    if $dry_run; then
      echo "  [DRY RUN] Would start tmux session: $session_name"
      echo "  [DRY RUN] Command: devcontainer exec ... claude -p \"...\" --dangerously-skip-permissions"
    else
      tmux kill-session -t "$session_name" 2>/dev/null || true

      tmux new-session -d -s "$session_name" \
        -e "DEV_KEY=$dev_key" \
        -e "DEV_WT_PATH=$wt_path" \
        -e "DEV_ISSUE_NUM=$issue_num" \
        'devcontainer exec --workspace-folder "$DEV_WT_PATH" --id-label "dev-worktree=$DEV_KEY" claude -p "You are an autonomous AI coding agent. Read .dev-issue.md for the GitHub Issue you need to resolve. Work autonomously: understand the issue, explore the codebase, implement the fix, run tests if available, then create a PR with gh pr create. Do not ask for confirmation — just do it." --dangerously-skip-permissions; echo ""; echo "--- Issue #${DEV_ISSUE_NUM} agent finished. Press Enter to close. ---"; read'

      echo "  Agent started (tmux session: $session_name)"
    fi
    echo ""
  done

  echo "Monitor with: dev dash  or  tmux attach -t dev-issue-<N>"
}

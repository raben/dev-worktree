#!/usr/bin/env bats
# Unit tests for lib/dev/session.sh

load test_helper

# ═══════════════════════════════════════════════════════════════
# _resolve_name_or_select
# ═══════════════════════════════════════════════════════════════

@test "_resolve_name_or_select: auto-selects single running container" {
  create_mock "docker" '
    if [[ "$*" == *"--format"* ]]; then
      echo "myproject/feature1"
    fi
  '
  # Cannot use `run` because DEV_SELECTED_KEY is set in the current shell.
  # Call directly and check the variable.
  _resolve_name_or_select
  [ "$DEV_SELECTED_KEY" = "myproject/feature1" ]
}

@test "_resolve_name_or_select: errors when no containers running" {
  # Mock docker to return empty output for ps queries
  create_mock "docker" 'exit 0'
  run _resolve_name_or_select
  [ "$status" -eq 1 ]
  # The error message may come from _resolve_name_or_select or _select_environment
  [[ "$output" == *"No running environments"* ]] || [[ "$output" == *"No dev-worktree environments found"* ]]
}

# ═══════════════════════════════════════════════════════════════
# cmd_code --help
# ═══════════════════════════════════════════════════════════════

@test "cmd_code --help: shows usage" {
  run cmd_code --help
  [ "$status" -eq 0 ]
  [[ "$output" == *"Usage: dev code"* ]]
  [[ "$output" == *"AI coding session"* ]]
}

@test "cmd_code -h: shows usage" {
  run cmd_code -h
  [ "$status" -eq 0 ]
  [[ "$output" == *"Usage: dev code"* ]]
}

# ═══════════════════════════════════════════════════════════════
# cmd_shell --help
# ═══════════════════════════════════════════════════════════════

@test "cmd_shell --help: shows usage" {
  run cmd_shell --help
  [ "$status" -eq 0 ]
  [[ "$output" == *"Usage: dev shell"* ]]
  [[ "$output" == *"Open a shell"* ]]
}

@test "cmd_shell -h: shows usage" {
  run cmd_shell -h
  [ "$status" -eq 0 ]
  [[ "$output" == *"Usage: dev shell"* ]]
}

# ═══════════════════════════════════════════════════════════════
# cmd_dash --help
# ═══════════════════════════════════════════════════════════════

@test "cmd_dash --help: shows usage" {
  run cmd_dash --help
  [ "$status" -eq 0 ]
  [[ "$output" == *"Usage: dev dash"* ]]
  [[ "$output" == *"tmux dashboard"* ]]
}

@test "cmd_dash -h: shows usage" {
  run cmd_dash -h
  [ "$status" -eq 0 ]
  [[ "$output" == *"Usage: dev dash"* ]]
}

# ═══════════════════════════════════════════════════════════════
# cmd_code: rejects unknown options
# ═══════════════════════════════════════════════════════════════

@test "cmd_code: rejects unknown option" {
  run cmd_code --unknown
  [ "$status" -eq 1 ]
  [[ "$output" == *"Unknown option"* ]]
}

# ═══════════════════════════════════════════════════════════════
# cmd_shell: rejects unknown options
# ═══════════════════════════════════════════════════════════════

@test "cmd_shell: rejects unknown option" {
  run cmd_shell --unknown
  [ "$status" -eq 1 ]
  [[ "$output" == *"Unknown option"* ]]
}

# ═══════════════════════════════════════════════════════════════
# cmd_dash: rejects unknown options
# ═══════════════════════════════════════════════════════════════

@test "cmd_dash: rejects unknown argument" {
  run cmd_dash --unknown
  [ "$status" -eq 1 ]
  [[ "$output" == *"Unknown option"* ]]
}

@test "cmd_dash: rejects positional argument" {
  run cmd_dash somename
  [ "$status" -eq 1 ]
  [[ "$output" == *"Unknown argument"* ]]
}

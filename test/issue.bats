#!/usr/bin/env bats
# Unit tests for lib/dev/issue.sh

load test_helper

# ═══════════════════════════════════════════════════════════════
# cmd_issue --help
# ═══════════════════════════════════════════════════════════════

@test "cmd_issue --help: shows usage" {
  run cmd_issue --help
  [ "$status" -eq 0 ]
  [[ "$output" == *"Usage: dev issue"* ]]
  [[ "$output" == *"GitHub Issues"* ]]
}

@test "cmd_issue -h: shows usage" {
  run cmd_issue -h
  [ "$status" -eq 0 ]
  [[ "$output" == *"Usage: dev issue"* ]]
}

@test "cmd_issue --help: documents --dry-run option" {
  run cmd_issue --help
  [ "$status" -eq 0 ]
  [[ "$output" == *"--dry-run"* ]]
}

@test "cmd_issue --help: mentions comma-separated numbers" {
  run cmd_issue --help
  [ "$status" -eq 0 ]
  [[ "$output" == *"Comma-separated"* ]]
}

# ═══════════════════════════════════════════════════════════════
# cmd_issue: rejects unknown options
# ═══════════════════════════════════════════════════════════════

@test "cmd_issue: rejects unknown option" {
  run cmd_issue --unknown
  [ "$status" -eq 1 ]
  [[ "$output" == *"Unknown option"* ]]
}

# ═══════════════════════════════════════════════════════════════
# cmd_issue --dry-run
# ═══════════════════════════════════════════════════════════════

@test "cmd_issue --dry-run 42: shows dry run header" {
  # Mock all required dependencies
  create_mock "gh" 'exit 0'
  create_mock "tmux" 'exit 0'
  create_mock "jq" 'exit 0'
  create_mock "git" 'echo "/tmp/fake-project"'
  # docker mock needed for _container_is_running / _resolve_key
  create_mock "docker" 'echo ""'

  run cmd_issue --dry-run 42
  [ "$status" -eq 0 ]
  [[ "$output" == *"DRY RUN"* ]]
}

@test "cmd_issue --dry-run 42: shows issue number" {
  create_mock "gh" 'exit 0'
  create_mock "tmux" 'exit 0'
  create_mock "jq" 'exit 0'
  create_mock "git" 'echo "/tmp/fake-project"'
  create_mock "docker" 'echo ""'

  run cmd_issue --dry-run 42
  [ "$status" -eq 0 ]
  [[ "$output" == *"Issue #42"* ]]
}

@test "cmd_issue --dry-run 42: shows would-run dev up" {
  create_mock "gh" 'exit 0'
  create_mock "tmux" 'exit 0'
  create_mock "jq" 'exit 0'
  create_mock "git" 'echo "/tmp/fake-project"'
  create_mock "docker" 'echo ""'

  run cmd_issue --dry-run 42
  [ "$status" -eq 0 ]
  [[ "$output" == *"dev up issue-42"* ]]
}

@test "cmd_issue --dry-run 42: shows tmux session name" {
  create_mock "gh" 'exit 0'
  create_mock "tmux" 'exit 0'
  create_mock "jq" 'exit 0'
  create_mock "git" 'echo "/tmp/fake-project"'
  create_mock "docker" 'echo ""'

  run cmd_issue --dry-run 42
  [ "$status" -eq 0 ]
  [[ "$output" == *"dev-issue-42"* ]]
}

# ═══════════════════════════════════════════════════════════════
# cmd_issue: invalid issue number
# ═══════════════════════════════════════════════════════════════

@test "cmd_issue: rejects non-numeric issue number" {
  create_mock "gh" 'exit 0'
  create_mock "tmux" 'exit 0'
  create_mock "jq" 'exit 0'

  run cmd_issue "abc"
  [ "$status" -eq 1 ]
  [[ "$output" == *"Invalid issue number"* ]]
}

@test "cmd_issue: accepts comma-separated issue numbers" {
  create_mock "gh" 'exit 0'
  create_mock "tmux" 'exit 0'
  create_mock "jq" 'exit 0'
  create_mock "git" 'echo "/tmp/fake-project"'
  create_mock "docker" 'echo ""'

  run cmd_issue --dry-run "12,34"
  [ "$status" -eq 0 ]
  [[ "$output" == *"Issue #12"* ]]
  [[ "$output" == *"Issue #34"* ]]
}

@test "cmd_issue: strips leading # from issue numbers" {
  create_mock "gh" 'exit 0'
  create_mock "tmux" 'exit 0'
  create_mock "jq" 'exit 0'
  create_mock "git" 'echo "/tmp/fake-project"'
  create_mock "docker" 'echo ""'

  run cmd_issue --dry-run "#42"
  [ "$status" -eq 0 ]
  [[ "$output" == *"Issue #42"* ]]
}

# ═══════════════════════════════════════════════════════════════
# cmd_issue: missing dependencies
# ═══════════════════════════════════════════════════════════════

@test "cmd_issue: fails when gh is missing" {
  # Create mocks for tmux and jq, but NOT gh.
  # Override PATH to only mock bin + system essentials (so real gh is not found).
  create_mock "tmux" 'exit 0'
  create_mock "jq" 'exit 0'
  local saved_path="$PATH"
  export PATH="${MOCK_BIN}:/usr/bin:/bin"

  run cmd_issue --dry-run 42
  export PATH="$saved_path"
  [ "$status" -eq 1 ]
  [[ "$output" == *"'gh' is required"* ]]
}

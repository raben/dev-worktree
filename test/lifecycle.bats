#!/usr/bin/env bats
# Unit tests for lib/dev/lifecycle.sh

load test_helper

# ═══════════════════════════════════════════════════════════════
# cmd_up --help
# ═══════════════════════════════════════════════════════════════

@test "cmd_up --help: shows usage" {
  run cmd_up --help
  [ "$status" -eq 0 ]
  [[ "$output" == *"Usage: dev up"* ]]
  [[ "$output" == *"Create (or resume)"* ]]
}

@test "cmd_up -h: shows usage" {
  run cmd_up -h
  [ "$status" -eq 0 ]
  [[ "$output" == *"Usage: dev up"* ]]
}

@test "cmd_up --help: documents --project-dir option" {
  run cmd_up --help
  [ "$status" -eq 0 ]
  [[ "$output" == *"--project-dir"* ]]
}

@test "cmd_up --help: documents base-branch argument" {
  run cmd_up --help
  [ "$status" -eq 0 ]
  [[ "$output" == *"base-branch"* ]]
}

# ═══════════════════════════════════════════════════════════════
# cmd_up: rejects unknown options
# ═══════════════════════════════════════════════════════════════

@test "cmd_up: rejects unknown option" {
  run cmd_up --unknown
  [ "$status" -eq 1 ]
  [[ "$output" == *"Unknown option"* ]]
}

# ═══════════════════════════════════════════════════════════════
# cmd_down --help
# ═══════════════════════════════════════════════════════════════

@test "cmd_down --help: shows usage" {
  run cmd_down --help
  [ "$status" -eq 0 ]
  [[ "$output" == *"Usage: dev down"* ]]
  [[ "$output" == *"Stop devcontainer"* ]]
}

@test "cmd_down -h: shows usage" {
  run cmd_down -h
  [ "$status" -eq 0 ]
  [[ "$output" == *"Usage: dev down"* ]]
}

@test "cmd_down: rejects unknown option" {
  run cmd_down --unknown
  [ "$status" -eq 1 ]
  [[ "$output" == *"Unknown option"* ]]
}

# ═══════════════════════════════════════════════════════════════
# cmd_prune --help
# ═══════════════════════════════════════════════════════════════

@test "cmd_prune --help: shows usage" {
  run cmd_prune --help
  [ "$status" -eq 0 ]
  [[ "$output" == *"Usage: dev prune"* ]]
  [[ "$output" == *"Remove worktree"* ]]
}

@test "cmd_prune -h: shows usage" {
  run cmd_prune -h
  [ "$status" -eq 0 ]
  [[ "$output" == *"Usage: dev prune"* ]]
}

@test "cmd_prune --help: mentions confirmation" {
  run cmd_prune --help
  [ "$status" -eq 0 ]
  [[ "$output" == *"confirmation"* ]]
}

@test "cmd_prune: rejects unknown option" {
  run cmd_prune --unknown
  [ "$status" -eq 1 ]
  [[ "$output" == *"Unknown option"* ]]
}

# ═══════════════════════════════════════════════════════════════
# cmd_list --help
# ═══════════════════════════════════════════════════════════════

@test "cmd_list --help: shows usage" {
  run cmd_list --help
  [ "$status" -eq 0 ]
  [[ "$output" == *"Usage: dev list"* ]]
  [[ "$output" == *"List dev-worktree environments"* ]]
}

@test "cmd_list -h: shows usage" {
  run cmd_list -h
  [ "$status" -eq 0 ]
  [[ "$output" == *"Usage: dev list"* ]]
}

@test "cmd_list --help: documents --json option" {
  run cmd_list --help
  [ "$status" -eq 0 ]
  [[ "$output" == *"--json"* ]]
}

@test "cmd_list --help: documents --project-dir option" {
  run cmd_list --help
  [ "$status" -eq 0 ]
  [[ "$output" == *"--project-dir"* ]]
}

@test "cmd_list: rejects unknown option" {
  run cmd_list --unknown
  [ "$status" -eq 1 ]
  [[ "$output" == *"Unknown option"* ]]
}

# ═══════════════════════════════════════════════════════════════
# cmd_list: no environments
# ═══════════════════════════════════════════════════════════════

@test "cmd_list: shows message when no environments found" {
  create_mock "docker" 'echo ""'
  run cmd_list
  [ "$status" -eq 0 ]
  [[ "$output" == *"No dev-worktree environments found"* ]]
}

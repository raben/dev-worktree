#!/usr/bin/env bats
# Unit tests for lib/dev/utils.sh

load test_helper

# ═══════════════════════════════════════════════════════════════
# _validate_name
# ═══════════════════════════════════════════════════════════════

@test "_validate_name: accepts simple alphanumeric name" {
  run _validate_name "feature1"
  [ "$status" -eq 0 ]
}

@test "_validate_name: accepts name with dots" {
  run _validate_name "release.1.0"
  [ "$status" -eq 0 ]
}

@test "_validate_name: accepts name with hyphens" {
  run _validate_name "my-feature"
  [ "$status" -eq 0 ]
}

@test "_validate_name: accepts name with slashes" {
  run _validate_name "project/feature"
  [ "$status" -eq 0 ]
}

@test "_validate_name: accepts name with underscores" {
  run _validate_name "my_feature"
  [ "$status" -eq 0 ]
}

@test "_validate_name: accepts mixed valid characters" {
  run _validate_name "proj/feat-1.2_rc"
  [ "$status" -eq 0 ]
}

@test "_validate_name: rejects empty string" {
  run _validate_name ""
  [ "$status" -eq 1 ]
  [[ "$output" == *"Invalid name"* ]]
}

@test "_validate_name: rejects name with spaces" {
  run _validate_name "my feature"
  [ "$status" -eq 1 ]
  [[ "$output" == *"Invalid name"* ]]
}

@test "_validate_name: rejects name starting with dot" {
  run _validate_name ".hidden"
  [ "$status" -eq 1 ]
  [[ "$output" == *"Invalid name"* ]]
}

@test "_validate_name: rejects name starting with hyphen" {
  run _validate_name "-flag"
  [ "$status" -eq 1 ]
  [[ "$output" == *"Invalid name"* ]]
}

@test "_validate_name: rejects name containing .." {
  run _validate_name "foo..bar"
  [ "$status" -eq 1 ]
  [[ "$output" == *"must not contain '..'"* ]]
}

@test "_validate_name: rejects name with special characters (semicolon)" {
  run _validate_name "foo;bar"
  [ "$status" -eq 1 ]
  [[ "$output" == *"Invalid name"* ]]
}

@test "_validate_name: rejects name with backtick" {
  run _validate_name 'foo`bar'
  [ "$status" -eq 1 ]
  [[ "$output" == *"Invalid name"* ]]
}

@test "_validate_name: rejects name with dollar sign" {
  run _validate_name 'foo$bar'
  [ "$status" -eq 1 ]
  [[ "$output" == *"Invalid name"* ]]
}

@test "_validate_name: rejects name with parentheses" {
  run _validate_name 'foo(bar)'
  [ "$status" -eq 1 ]
  [[ "$output" == *"Invalid name"* ]]
}

# ═══════════════════════════════════════════════════════════════
# _validate_exec_cmd
# ═══════════════════════════════════════════════════════════════

@test "_validate_exec_cmd: accepts 'claude'" {
  run _validate_exec_cmd "claude"
  [ "$status" -eq 0 ]
}

@test "_validate_exec_cmd: accepts 'claude --dangerously-skip-permissions'" {
  run _validate_exec_cmd "claude --dangerously-skip-permissions"
  [ "$status" -eq 0 ]
}

@test "_validate_exec_cmd: accepts command with path separators" {
  run _validate_exec_cmd "/usr/bin/claude"
  [ "$status" -eq 0 ]
}

@test "_validate_exec_cmd: accepts command with equals and colons" {
  run _validate_exec_cmd "ENV_VAR=value:other cmd"
  [ "$status" -eq 0 ]
}

@test "_validate_exec_cmd: accepts command with @ and commas" {
  run _validate_exec_cmd "user@host,flag"
  [ "$status" -eq 0 ]
}

@test "_validate_exec_cmd: rejects command with semicolon" {
  run _validate_exec_cmd "claude; rm -rf /"
  [ "$status" -eq 1 ]
  [[ "$output" == *"unsafe characters"* ]]
}

@test "_validate_exec_cmd: rejects command with pipe" {
  run _validate_exec_cmd "claude | cat"
  [ "$status" -eq 1 ]
  [[ "$output" == *"unsafe characters"* ]]
}

@test "_validate_exec_cmd: rejects command with backtick" {
  run _validate_exec_cmd 'claude `whoami`'
  [ "$status" -eq 1 ]
  [[ "$output" == *"unsafe characters"* ]]
}

@test "_validate_exec_cmd: rejects command with dollar-paren" {
  run _validate_exec_cmd 'claude $(whoami)'
  [ "$status" -eq 1 ]
  [[ "$output" == *"unsafe characters"* ]]
}

@test "_validate_exec_cmd: rejects command with ampersand" {
  run _validate_exec_cmd "claude & echo pwned"
  [ "$status" -eq 1 ]
  [[ "$output" == *"unsafe characters"* ]]
}

@test "_validate_exec_cmd: rejects command with redirect" {
  run _validate_exec_cmd "claude > /etc/passwd"
  [ "$status" -eq 1 ]
  [[ "$output" == *"unsafe characters"* ]]
}

# ═══════════════════════════════════════════════════════════════
# _trim
# ═══════════════════════════════════════════════════════════════

@test "_trim: removes leading spaces" {
  result=$(_trim "   hello")
  [ "$result" = "hello" ]
}

@test "_trim: removes trailing spaces" {
  result=$(_trim "hello   ")
  [ "$result" = "hello" ]
}

@test "_trim: removes leading and trailing spaces" {
  result=$(_trim "   hello   ")
  [ "$result" = "hello" ]
}

@test "_trim: removes tabs" {
  result=$(_trim "$(printf '\thello\t')")
  [ "$result" = "hello" ]
}

@test "_trim: preserves internal spaces" {
  result=$(_trim "  hello world  ")
  [ "$result" = "hello world" ]
}

@test "_trim: handles string with no whitespace" {
  result=$(_trim "hello")
  [ "$result" = "hello" ]
}

@test "_trim: handles empty string" {
  result=$(_trim "")
  [ "$result" = "" ]
}

@test "_trim: handles whitespace-only string" {
  result=$(_trim "   ")
  [ "$result" = "" ]
}

# ═══════════════════════════════════════════════════════════════
# _allocate_ports
# ═══════════════════════════════════════════════════════════════

@test "_allocate_ports: allocates ports from .env.example" {
  local env_file="${TEST_TMPDIR}/.env.example"
  cat > "$env_file" <<'EOF'
WT_NAME=myproject
COMPOSE_PROJECT_NAME=myproject
WT_APP_PORT=3000
WT_DB_PORT=5432
EOF

  run _allocate_ports "$env_file"
  [ "$status" -eq 0 ]
  [[ "$output" == *"WT_NAME=myproject"* ]]
  [[ "$output" == *"COMPOSE_PROJECT_NAME=myproject"* ]]
  [[ "$output" == *"WT_APP_PORT="* ]]
  [[ "$output" == *"WT_DB_PORT="* ]]
}

@test "_allocate_ports: passes through non-port variables unchanged" {
  local env_file="${TEST_TMPDIR}/.env.example"
  cat > "$env_file" <<'EOF'
WT_NAME=testproject
SOME_VAR=hello
EOF

  run _allocate_ports "$env_file"
  [ "$status" -eq 0 ]
  [[ "$output" == *"WT_NAME=testproject"* ]]
  [[ "$output" == *"SOME_VAR=hello"* ]]
}

@test "_allocate_ports: rejects non-numeric port value" {
  local env_file="${TEST_TMPDIR}/.env.example"
  cat > "$env_file" <<'EOF'
WT_APP_PORT=abc
EOF

  run _allocate_ports "$env_file"
  [ "$status" -eq 1 ]
  [[ "$output" == *"Invalid port value"* ]]
}

@test "_allocate_ports: rejects port below 1024" {
  local env_file="${TEST_TMPDIR}/.env.example"
  cat > "$env_file" <<'EOF'
WT_APP_PORT=80
EOF

  run _allocate_ports "$env_file"
  [ "$status" -eq 1 ]
  [[ "$output" == *"out of valid range"* ]]
}

@test "_allocate_ports: rejects port above 65535" {
  local env_file="${TEST_TMPDIR}/.env.example"
  cat > "$env_file" <<'EOF'
WT_APP_PORT=70000
EOF

  run _allocate_ports "$env_file"
  [ "$status" -eq 1 ]
  [[ "$output" == *"out of valid range"* ]]
}

@test "_allocate_ports: skips comment lines" {
  local env_file="${TEST_TMPDIR}/.env.example"
  cat > "$env_file" <<'EOF'
# This is a comment
WT_NAME=testproject
  # Indented comment
WT_APP_PORT=3000
EOF

  run _allocate_ports "$env_file"
  [ "$status" -eq 0 ]
  # Should only have WT_NAME and WT_APP_PORT lines, no comment output
  [[ "$output" == *"WT_NAME=testproject"* ]]
  [[ "$output" == *"WT_APP_PORT="* ]]
}

@test "_allocate_ports: port value is numeric in output" {
  local env_file="${TEST_TMPDIR}/.env.example"
  cat > "$env_file" <<'EOF'
WT_APP_PORT=8080
EOF

  run _allocate_ports "$env_file"
  [ "$status" -eq 0 ]
  # Extract port value and verify it is numeric
  local port_val
  port_val=$(echo "$output" | grep 'WT_APP_PORT=' | cut -d= -f2)
  [[ "$port_val" =~ ^[0-9]+$ ]]
}

# ═══════════════════════════════════════════════════════════════
# _port_is_available
# ═══════════════════════════════════════════════════════════════

@test "_port_is_available: high random port is available" {
  # Use a high port unlikely to be in use
  local port=59132
  run _port_is_available "$port"
  [ "$status" -eq 0 ]
}

@test "_port_is_available: returns success for unreachable port" {
  # Port 64999 is very unlikely to be occupied
  run _port_is_available 64999
  [ "$status" -eq 0 ]
}

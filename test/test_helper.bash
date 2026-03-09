# test_helper.bash — shared setup for dev-worktree bats tests
# Sourced by each .bats file via: load test_helper

# ─── Project paths ────────────────────────────────────────────
# BATS_TEST_DIRNAME is set by bats to the directory containing the .bats file.
# Resolve PROJECT_ROOT relative to this helper file's own location.
_test_helper_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
export PROJECT_ROOT="$(cd "${_test_helper_dir}/.." && pwd)"
export DEV_LIB_DIR="${PROJECT_ROOT}/lib/dev"
unset _test_helper_dir

# ─── Globals expected by lib files ────────────────────────────
export VERSION="0.0.0-test"

# ─── Preserve original PATH for teardown ──────────────────────
_ORIG_PATH="$PATH"

# ─── Temp directory for test artifacts ────────────────────────
setup() {
  TEST_TMPDIR="$(mktemp -d "${BATS_TMPDIR:-/tmp}/dev-worktree-test-XXXXXX")"
  export TEST_TMPDIR

  # Create mock bin directory and prepend to PATH so tests can
  # provide fake docker/git/etc. commands
  MOCK_BIN="${TEST_TMPDIR}/mock-bin"
  mkdir -p "$MOCK_BIN"
  export MOCK_BIN
  export PATH="${MOCK_BIN}:${_ORIG_PATH}"
}

teardown() {
  export PATH="$_ORIG_PATH"
  if [ -d "${TEST_TMPDIR:-}" ]; then
    rm -rf "$TEST_TMPDIR"
  fi
}

# ─── Source library files ─────────────────────────────────────
# bats uses its own error handling; the lib files use `set -euo pipefail`
# via bin/dev. We source them without that strict mode so bats `run`
# can capture non-zero exits properly.
#
# Source order matters: utils.sh first (others depend on it).
source "${DEV_LIB_DIR}/utils.sh"
source "${DEV_LIB_DIR}/lifecycle.sh"
source "${DEV_LIB_DIR}/session.sh"
source "${DEV_LIB_DIR}/issue.sh"
source "${DEV_LIB_DIR}/init.sh"
source "${DEV_LIB_DIR}/doctor.sh"

# ─── Mock helpers ─────────────────────────────────────────────

# Create a mock executable in MOCK_BIN
# Usage: create_mock <name> <script_body>
create_mock() {
  local name="$1"
  local body="${2:-exit 0}"
  cat > "${MOCK_BIN}/${name}" <<SCRIPT
#!/usr/bin/env bash
${body}
SCRIPT
  chmod +x "${MOCK_BIN}/${name}"
}

# Create a docker mock that returns specified output
# Usage: mock_docker <output> [exit_code]
mock_docker() {
  local output="${1:-}"
  local exit_code="${2:-0}"
  create_mock "docker" "echo '${output}'; exit ${exit_code}"
}

# Create a git mock
# Usage: mock_git <output> [exit_code]
mock_git() {
  local output="${1:-}"
  local exit_code="${2:-0}"
  create_mock "git" "echo '${output}'; exit ${exit_code}"
}

#!/usr/bin/env bash
# E2E test for dev-worktree Go CLI
# Tests: doctor, up, list, list --json, exec, down, up (resume), prune, list (empty)
set -euo pipefail

DEV_BIN="/tmp/dev-e2e-test/dev"
export PATH="/tmp/dev-e2e-test:$PATH"

PASS=0
FAIL=0
TOTAL=0
TEST_DIR=""

# ── Helpers ─────────────────────────────────────────────────────────────────

pass() {
  PASS=$((PASS + 1))
  TOTAL=$((TOTAL + 1))
  echo "PASS: $1"
}

fail() {
  FAIL=$((FAIL + 1))
  TOTAL=$((TOTAL + 1))
  echo "FAIL: $1"
  if [[ -n "${2:-}" ]]; then
    echo "      $2"
  fi
}

cleanup() {
  echo ""
  echo "=== Cleanup ==="
  # Remove any dev-worktree containers created during test
  local ids
  ids=$(docker ps -aq --filter "label=dev-worktree" 2>/dev/null || true)
  if [[ -n "$ids" ]]; then
    echo "Removing leftover containers..."
    docker rm -f $ids >/dev/null 2>&1 || true
  fi
  # Remove test directory
  if [[ -n "$TEST_DIR" && -d "$TEST_DIR" ]]; then
    # Clean up worktrees created under TEST_DIR's parent
    local parent
    parent=$(dirname "$TEST_DIR")
    if [[ -d "$parent/dev" ]]; then
      # Force remove any worktrees that point into our temp dir
      (cd "$TEST_DIR" && git worktree list --porcelain 2>/dev/null | grep "^worktree " | awk '{print $2}' | while read -r wt; do
        [[ "$wt" != "$TEST_DIR" ]] && git worktree remove --force "$wt" 2>/dev/null || true
      done) || true
      rm -rf "$parent/dev" 2>/dev/null || true
    fi
    rm -rf "$TEST_DIR"
    echo "Removed test directory."
  fi
}

trap cleanup EXIT

# ── Setup ───────────────────────────────────────────────────────────────────

echo "=== Setup ==="

if [[ ! -x "$DEV_BIN" ]]; then
  echo "ERROR: dev binary not found at $DEV_BIN"
  exit 1
fi

TEST_DIR=$(mktemp -d)
echo "Test directory: $TEST_DIR"

cd "$TEST_DIR"
git init -b main >/dev/null 2>&1
git commit --allow-empty -m "init" >/dev/null 2>&1

cat > .dev.yml << 'EOF'
image: node:22-slim
ports: [3000]
EOF

git add . && git commit -m "add dev.yml" >/dev/null 2>&1

echo "Setup complete."
echo ""

# ── Test 1: dev doctor ──────────────────────────────────────────────────────

echo "=== Test 1: dev doctor ==="
DOCTOR_OUT=$("$DEV_BIN" doctor 2>&1) || true

if echo "$DOCTOR_OUT" | grep -q "Docker daemon"; then
  pass "dev doctor - detects Docker"
else
  fail "dev doctor - detects Docker" "Output: $DOCTOR_OUT"
fi

if echo "$DOCTOR_OUT" | grep -q "Git repository"; then
  pass "dev doctor - detects git"
else
  fail "dev doctor - detects git" "Output: $DOCTOR_OUT"
fi

if echo "$DOCTOR_OUT" | grep -q ".dev.yml"; then
  pass "dev doctor - detects .dev.yml"
else
  fail "dev doctor - detects .dev.yml" "Output: $DOCTOR_OUT"
fi

# ── Test 2: dev up test-branch ──────────────────────────────────────────────

echo ""
echo "=== Test 2: dev up test-branch ==="
UP_OUT=$("$DEV_BIN" up test-branch 2>&1) || {
  # The 'up' command installs claude code which will fail in node:22-slim without npm.
  # That's expected behavior - let's check if at least the container+worktree were created.
  true
}
echo "$UP_OUT"

# Check container exists with dev-worktree label
PROJECT_NAME=$(basename "$TEST_DIR")
DEV_KEY="$PROJECT_NAME/test-branch"
CONTAINER_NAME="$PROJECT_NAME-test-branch"

CONTAINER_ID=$(docker ps -aq --filter "label=dev-worktree=$DEV_KEY" 2>/dev/null | head -1)

if [[ -n "$CONTAINER_ID" ]]; then
  CONTAINER_STATE=$(docker inspect --format '{{.State.Status}}' "$CONTAINER_ID" 2>/dev/null || echo "unknown")
  if [[ "$CONTAINER_STATE" == "running" ]]; then
    pass "dev up - container is running"
  else
    fail "dev up - container is running" "State: $CONTAINER_STATE"
  fi
else
  fail "dev up - container is running" "No container found for key $DEV_KEY"
fi

# Check worktree directory exists
WT_PARENT=$(dirname "$TEST_DIR")
WT_PATH="$WT_PARENT/dev/$PROJECT_NAME/test-branch"

if [[ -d "$WT_PATH" ]]; then
  pass "dev up - worktree directory exists"
else
  fail "dev up - worktree directory exists" "Expected $WT_PATH"
fi

# ── Test 3: dev list ────────────────────────────────────────────────────────

echo ""
echo "=== Test 3: dev list ==="
LIST_OUT=$("$DEV_BIN" list 2>&1) || true
echo "$LIST_OUT"

if echo "$LIST_OUT" | grep -q "running"; then
  pass "dev list - shows running status"
else
  fail "dev list - shows running status" "Output: $LIST_OUT"
fi

# ── Test 4: dev list --json ─────────────────────────────────────────────────

echo ""
echo "=== Test 4: dev list --json ==="
JSON_OUT=$("$DEV_BIN" list --json 2>&1) || true

# Validate JSON
if echo "$JSON_OUT" | python3 -m json.tool >/dev/null 2>&1; then
  pass "dev list --json - valid JSON output"
else
  fail "dev list --json - valid JSON output" "Output: $JSON_OUT"
fi

# ── Test 5: container exec (shell substitute) ──────────────────────────────

echo ""
echo "=== Test 5: container exec ==="
if [[ -n "$CONTAINER_ID" ]]; then
  EXEC_OUT=$(docker exec "$CONTAINER_ID" echo hello 2>&1) || true
  if [[ "$EXEC_OUT" == "hello" ]]; then
    pass "container exec - echo hello works"
  else
    fail "container exec - echo hello works" "Output: $EXEC_OUT"
  fi
else
  fail "container exec - echo hello works" "No container to exec into"
fi

# ── Test 6: dev down test-branch ───────────────────────────────────────────

echo ""
echo "=== Test 6: dev down test-branch ==="
DOWN_OUT=$("$DEV_BIN" down test-branch 2>&1) || true
echo "$DOWN_OUT"

if [[ -n "$CONTAINER_ID" ]]; then
  CONTAINER_STATE=$(docker inspect --format '{{.State.Status}}' "$CONTAINER_ID" 2>/dev/null || echo "removed")
  if [[ "$CONTAINER_STATE" == "exited" ]]; then
    pass "dev down - container stopped"
  else
    fail "dev down - container stopped" "State: $CONTAINER_STATE"
  fi
else
  fail "dev down - container stopped" "No container found"
fi

# ── Test 7: dev up test-branch (resume) ────────────────────────────────────

echo ""
echo "=== Test 7: dev up test-branch (resume) ==="
UP2_OUT=$("$DEV_BIN" up test-branch 2>&1) || true
echo "$UP2_OUT"

# After resume, a new or restarted container should be running
# The old container was stopped; 'up' creates a new one since the old is stopped
CONTAINER_ID2=$(docker ps -q --filter "label=dev-worktree=$DEV_KEY" 2>/dev/null | head -1)

if [[ -n "$CONTAINER_ID2" ]]; then
  CONTAINER_STATE2=$(docker inspect --format '{{.State.Status}}' "$CONTAINER_ID2" 2>/dev/null || echo "unknown")
  if [[ "$CONTAINER_STATE2" == "running" ]]; then
    pass "dev up (resume) - container is running again"
  else
    fail "dev up (resume) - container is running again" "State: $CONTAINER_STATE2"
  fi
else
  fail "dev up (resume) - container is running again" "No running container found"
fi

# ── Test 8: dev prune test-branch ──────────────────────────────────────────

echo ""
echo "=== Test 8: dev prune test-branch ==="
# Feed "y" for confirmation (may ask twice if dirty)
PRUNE_OUT=$(printf 'y\ny\n' | "$DEV_BIN" prune test-branch 2>&1) || true
echo "$PRUNE_OUT"

# Verify container removed
REMAINING=$(docker ps -aq --filter "label=dev-worktree=$DEV_KEY" 2>/dev/null)
if [[ -z "$REMAINING" ]]; then
  pass "dev prune - container removed"
else
  fail "dev prune - container removed" "Containers still exist: $REMAINING"
fi

# Verify worktree gone
if [[ ! -d "$WT_PATH" ]]; then
  pass "dev prune - worktree removed"
else
  fail "dev prune - worktree removed" "Directory still exists: $WT_PATH"
fi

# ── Test 9: dev list (empty) ───────────────────────────────────────────────

echo ""
echo "=== Test 9: dev list (empty) ==="
LIST_EMPTY=$("$DEV_BIN" list 2>&1) || true
echo "$LIST_EMPTY"

if echo "$LIST_EMPTY" | grep -qi "no environments"; then
  pass "dev list (empty) - shows no environments"
else
  fail "dev list (empty) - shows no environments" "Output: $LIST_EMPTY"
fi

# ── Summary ─────────────────────────────────────────────────────────────────

echo ""
echo "======================================="
echo "$PASS/$TOTAL tests passed"
if [[ $FAIL -gt 0 ]]; then
  echo "$FAIL test(s) FAILED"
  exit 1
else
  echo "All tests passed."
  exit 0
fi

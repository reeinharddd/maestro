#!/usr/bin/env bash
BIN="./okit"
DB="/tmp/opencode/okit-test-suite.db"
PASS=0
FAIL=0

green() { echo -e "\033[0;32m  PASS: $1\033[0m"; ((PASS++)); }
red() { echo -e "\033[0;31m  FAIL: $1\033[0m"; ((FAIL++)); }
run() {
  local name="$1"; shift
  local out; out=$(mktemp)
  if "$@" > "$out" 2>&1; then
    green "$name"
  else
    local ec=$?
    red "$name (exit=$ec)"
  fi
  rm -f "$out"
}
run_fail() {
  local name="$1"; shift
  local out; out=$(mktemp)
  if "$@" > "$out" 2>&1; then
    red "$name (should have failed)"
  else
    green "$name"
  fi
  rm -f "$out"
}

cleanup() { rm -f "$DB" /tmp/opencode/opencode.jsonc; }

echo "=== opencode-kit Test Suite ==="
echo ""

# 1
echo "--- Group 1: Basic ---"
run "help" $BIN --help
run "completion" $BIN completion

# 2
echo "--- Group 2: Empty DB ---"
cleanup
run "status empty" $BIN status --db "$DB"
run "providers list empty" $BIN providers list --db "$DB"
run "models list empty" $BIN models list --db "$DB"
run "route empty" $BIN route --db "$DB"
run "heal empty" $BIN heal --db "$DB"
run "query basic" $BIN query --db "$DB" "SELECT 1 AS test"
run "generate config empty" $BIN generate config --db "$DB"
run "generate agents empty" $BIN generate agents --db "$DB"
run "generate commands empty" $BIN generate commands --db "$DB"

run_fail "invalid table" $BIN query --db "$DB" "SELECT * FROM nonexistent"
run_fail "invalid SQL" $BIN query --db "$DB" "NOT SQL"
run_fail "nonexistent db" $BIN status --db "/nonexistent/deep/db.db"

# 3
echo "--- Group 3: Discover ---"
cleanup
run "discover" $BIN discover --db "$DB"

# 4
echo "--- Group 4: Audit ---"
run "audit" $BIN audit --db "$DB"

# 5
echo "--- Group 5: Models ---"
run "models list" $BIN models list --db "$DB"
run "models search codestral" $BIN models search --db "$DB" codestral
run "providers list" $BIN providers list --db "$DB"
run "providers add" $BIN providers add --db "$DB" --id "test" --api-base "https://test.com/v1" --key-env "TEST_KEY"

# 6
echo "--- Group 6: Intelligence ---"
run "profile" $BIN profile --db "$DB"
run "route reassign" $BIN route --reassign --db "$DB"
run "route show" $BIN route --db "$DB"
run "route task" $BIN route --task coding_fast --db "$DB"
run "heal" $BIN heal --db "$DB"

# 7
echo "--- Group 7: Generate ---"
run "generate config" $BIN generate config --db "$DB"
run "generate agents" $BIN generate agents --db "$DB"
run "generate commands" $BIN generate commands --db "$DB"

if [ -f /tmp/opencode/opencode.jsonc ]; then
  green "config file"
  python3 -c "import json; json.load(open('/tmp/opencode/opencode.jsonc'))" 2>/dev/null && green "valid JSON" || red "invalid JSON"
fi

# 8
echo "--- Group 8: Daily ---"
cleanup
run "daily" $BIN daily --db "$DB"
run "daily idempotent" $BIN daily --db "$DB"

# 9
echo "--- Group 9: Edge ---"
run "sync" $BIN sync --db "$DB"
run "sources list" $BIN sources list --db "$DB"

run_fail "empty query" $BIN query --db "$DB" ""

# Results
cleanup
echo ""
echo "=============================="
echo "  $PASS passed, $FAIL failed"
echo "=============================="
[ "$FAIL" -eq 0 ] && echo "  ALL TESTS PASSED" || echo "  Some tests failed"
exit $FAIL

#!/usr/bin/env bash
source "$(dirname "$0")/../lib.sh"

echo "help / usage output"

# No args → exit 0, shows usage (wrapper handles this before Docker)
output=$(run_vc 2>&1) && rc=0 || rc=$?
assert_exit 0 "$rc" "no args exits 0"
assert_output_contains "$output" "vc" "no args shows usage"

# --help → exit 0
output=$(run_vc --help 2>&1) && rc=0 || rc=$?
assert_exit 0 "$rc" "--help exits 0"
assert_output_contains "$output" "Usage:" "--help shows usage"

# -h → exit 0
output=$(run_vc -h 2>&1) && rc=0 || rc=$?
assert_exit 0 "$rc" "-h exits 0"

# help subcommand → exit 0
output=$(run_vc help 2>&1) && rc=0 || rc=$?
assert_exit 0 "$rc" "help subcommand exits 0"

# Usage text includes all three subcommands
assert_output_contains "$output" "[Cc]ompress" "usage mentions compress"
assert_output_contains "$output" "[Cc]leanup" "usage mentions cleanup"
assert_output_contains "$output" "[Aa]ssess" "usage mentions assess"

summarise

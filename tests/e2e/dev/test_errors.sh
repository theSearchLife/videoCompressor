#!/usr/bin/env bash
source "$(dirname "$0")/../lib.sh"

echo "error handling"

# --- Missing input directory ---

output=$(run_vc /tmp/nonexistent_dir_e2e_test --strategy balanced --suffix "_test" 2>&1) && rc=0 || rc=$?
assert_exit 1 "$rc" "missing dir exits non-zero"

# --- Invalid strategy ---

setup_tmp
dir="$TEST_DIR/err_strategy"
mkdir -p "$dir"
place_video "$dir/clip.mp4"
output=$(run_vc "$dir" --strategy bogus --suffix "_test" 2>&1) && rc=0 || rc=$?
assert_exit 1 "$rc" "invalid strategy exits non-zero"
assert_output_contains "$output" "unknown strategy" "invalid strategy error message"

# --- Invalid resolution ---

output=$(run_vc "$dir" --strategy balanced --resolution 480p --suffix "_test" 2>&1) && rc=0 || rc=$?
assert_exit 1 "$rc" "invalid resolution exits non-zero"
assert_output_contains "$output" "unknown resolution" "invalid resolution error message"

# --- Invalid audio mode ---

output=$(run_vc "$dir" --strategy balanced --audio ultrahigh --suffix "_test" 2>&1) && rc=0 || rc=$?
assert_exit 1 "$rc" "invalid audio exits non-zero"
assert_output_contains "$output" "unknown audio mode" "invalid audio error message"

# --- Invalid fps ---

output=$(run_vc "$dir" --strategy balanced --fps abc --suffix "_test" 2>&1) && rc=0 || rc=$?
assert_exit 1 "$rc" "invalid fps exits non-zero"
assert_output_contains "$output" "invalid fps" "invalid fps error message"

# --- Invalid skip-converted ---

output=$(run_vc "$dir" --strategy balanced --skip-converted maybe --suffix "_test" 2>&1) && rc=0 || rc=$?
assert_exit 1 "$rc" "invalid skip-converted exits non-zero"
assert_output_contains "$output" "invalid --skip-converted" "invalid skip-converted error message"

# --- Compress subcommand with no dir ---

output=$(run_vc compress 2>&1) && rc=0 || rc=$?
assert_exit 1 "$rc" "compress with no dir exits non-zero"

# --- Cleanup subcommand with no dir ---

output=$(run_vc cleanup 2>&1) && rc=0 || rc=$?
assert_exit 1 "$rc" "cleanup with no dir exits non-zero"

# --- Assess subcommand with no dir ---

output=$(run_vc assess 2>&1) && rc=0 || rc=$?
assert_exit 1 "$rc" "assess with no dir exits non-zero"

summarise

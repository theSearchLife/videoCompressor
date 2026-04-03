#!/usr/bin/env bash
# Post-build smoke tests.
# Verify the published GHCR image works for core workflows.
source "$(dirname "$0")/../lib.sh"
setup_tmp

echo "post-build smoke tests"

# --- Help ---

output=$(run_vc --help 2>&1) && rc=0 || rc=$?
assert_exit 0 "$rc" "help works"
assert_output_contains "$output" "Usage:" "help shows usage"

# --- Basic compress ---

dir="$TEST_DIR/smoke_compress"
mkdir -p "$dir"
place_video "$dir/clip.mp4"

run_vc "$dir" --strategy balanced --suffix "_smoke" >/dev/null 2>&1 && rc=0 || rc=$?
assert_exit 0 "$rc" "compress exits 0"
assert_file_exists "$dir/clip_smoke.mp4" "compress produces output"
assert_file_nonzero "$dir/clip_smoke.mp4" "compress output is non-empty"

# --- Dry-run ---

dir="$TEST_DIR/smoke_dryrun"
mkdir -p "$dir"
place_video "$dir/clip.mp4"

output=$(run_vc "$dir" --strategy balanced --suffix "_test" --dry-run 2>&1) && rc=0 || rc=$?
assert_exit 0 "$rc" "dry-run exits 0"
assert_output_contains "$output" "Would encode" "dry-run shows plan"
assert_file_not_exists "$dir/clip_test.mp4" "dry-run does not encode"

# --- Cleanup ---

dir="$TEST_DIR/smoke_cleanup"
mkdir -p "$dir"
place_video "$dir/clip.mp4"

run_vc "$dir" --strategy balanced --suffix "_smoke" >/dev/null 2>&1
assert_file_exists "$dir/clip_smoke.mp4" "compress for cleanup test produced output"

output=$(run_vc cleanup "$dir" --suffix "_smoke" 2>&1) && rc=0 || rc=$?
assert_exit 0 "$rc" "cleanup exits 0"
assert_file_exists "$dir/clip.mp4" "cleanup renamed output"
assert_file_not_exists "$dir/clip_smoke.mp4" "cleanup removed suffixed file"

# --- Error: missing dir ---

output=$(run_vc /tmp/nonexistent_e2e_smoke --strategy balanced 2>&1) && rc=0 || rc=$?
assert_exit 1 "$rc" "missing dir exits non-zero"

summarise

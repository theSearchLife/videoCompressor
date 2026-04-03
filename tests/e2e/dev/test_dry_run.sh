#!/usr/bin/env bash
source "$(dirname "$0")/../lib.sh"
setup_tmp

echo "dry-run mode"

dir="$TEST_DIR/dryrun"
mkdir -p "$dir"
place_video "$dir/clip.mp4"

output=$(run_vc "$dir" --strategy balanced --suffix "_test" --dry-run 2>&1) && rc=0 || rc=$?
assert_exit 0 "$rc" "dry-run exits 0"
assert_output_contains "$output" "Would encode" "dry-run shows plan"
assert_output_contains "$output" "clip" "dry-run lists the file"
assert_file_not_exists "$dir/clip_test.mp4" "dry-run does not create output file"

# Dry-run with multiple files
dir2="$TEST_DIR/dryrun_multi"
mkdir -p "$dir2/sub"
place_video "$dir2/a.mp4"
place_video "$dir2/sub/b.mp4"

output=$(run_vc "$dir2" --strategy balanced --suffix "_test" --dry-run 2>&1) && rc=0 || rc=$?
assert_exit 0 "$rc" "dry-run multi exits 0"
assert_output_contains "$output" "Would encode 2 files" "dry-run counts both files"

summarise

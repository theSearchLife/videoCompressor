#!/usr/bin/env bash
source "$(dirname "$0")/../lib.sh"
setup_tmp

echo "cleanup workflow"

# --- Basic compress then cleanup ---

dir="$TEST_DIR/cleanup_basic"
mkdir -p "$dir"
place_video "$dir/clip.mp4"

# Compress first
run_vc "$dir" --strategy balanced --suffix "_compressed" >/dev/null 2>&1
assert_file_exists "$dir/clip_compressed.mp4" "compress produced output for cleanup test"

# Run cleanup (non-interactive skips confirmation)
output=$(run_vc cleanup "$dir" --suffix "_compressed" 2>&1) && rc=0 || rc=$?
assert_exit 0 "$rc" "cleanup exits 0"
assert_file_not_exists "$dir/clip_compressed.mp4" "cleanup removed suffixed file"
assert_file_exists "$dir/clip.mp4" "cleanup renamed output to base name"

# --- Cleanup with custom suffix ---

dir="$TEST_DIR/cleanup_custom"
mkdir -p "$dir"
place_video "$dir/video.mp4"

run_vc "$dir" --strategy balanced --suffix "_v1" >/dev/null 2>&1
assert_file_exists "$dir/video_v1.mp4" "compress with _v1 suffix produced output"

output=$(run_vc cleanup "$dir" --suffix "_v1" 2>&1) && rc=0 || rc=$?
assert_exit 0 "$rc" "cleanup with custom suffix exits 0"
assert_file_not_exists "$dir/video_v1.mp4" "cleanup removed suffixed file"
assert_file_exists "$dir/video.mp4" "cleanup preserved base name"

# --- Cleanup with no matches ---

dir="$TEST_DIR/cleanup_nomatch"
mkdir -p "$dir"
place_video "$dir/clip.mp4"

output=$(run_vc cleanup "$dir" --suffix "_nonexistent" 2>&1) && rc=0 || rc=$?
assert_exit 0 "$rc" "cleanup no-match exits 0"
assert_output_contains "$output" "No converted outputs" "cleanup reports no matches"
assert_file_exists "$dir/clip.mp4" "original untouched when no matches"

summarise

#!/usr/bin/env bash
source "$(dirname "$0")/../lib.sh"
setup_tmp

echo "directory scanning"

# --- Recursive scan finds videos in subdirectories ---

dir="$TEST_DIR/recursive"
place_video "$dir/root.mp4"
place_video "$dir/sub1/a.mp4"
place_video "$dir/sub1/sub2/b.mp4"

output=$(run_vc "$dir" --strategy balanced --suffix "_test" --dry-run 2>&1) && rc=0 || rc=$?
assert_exit 0 "$rc" "recursive scan exits 0"
assert_output_contains "$output" "Would encode 3 files" "recursive scan finds all 3 videos"

# --- Mixed content: only picks video files ---

dir="$TEST_DIR/mixed"
place_video "$dir/videos/clip.mp4"
place_file "$dir/photos/IMG_001.jpg"
place_file "$dir/photos/IMG_002.png"
place_file "$dir/docs/report.pdf"
place_file "$dir/docs/notes.txt"
place_file "$dir/data.xlsx"

output=$(run_vc "$dir" --strategy balanced --suffix "_test" --dry-run 2>&1) && rc=0 || rc=$?
assert_exit 0 "$rc" "mixed content scan exits 0"
assert_output_contains "$output" "Would encode 1 file" "mixed content finds only the video"

# --- Unicode filename ---

dir="$TEST_DIR/unicode"
place_video "$dir/видео клип.mp4"

output=$(run_vc "$dir" --strategy balanced --suffix "_test" --dry-run 2>&1) && rc=0 || rc=$?
assert_exit 0 "$rc" "unicode filename scan exits 0"
assert_output_contains "$output" "Would encode 1 file" "unicode filename found"

# --- Empty directory ---

dir="$TEST_DIR/empty"
mkdir -p "$dir"

output=$(run_vc "$dir" --strategy balanced --suffix "_test" 2>&1) && rc=0 || rc=$?
assert_exit 0 "$rc" "empty dir exits 0"
assert_output_contains "$output" "No video files found" "empty dir reports no videos"

# --- Directory with only non-video files ---

dir="$TEST_DIR/no_videos"
place_file "$dir/photo.jpg"
place_file "$dir/document.pdf"
place_file "$dir/music.mp3"

output=$(run_vc "$dir" --strategy balanced --suffix "_test" 2>&1) && rc=0 || rc=$?
assert_exit 0 "$rc" "no-videos dir exits 0"
assert_output_contains "$output" "No video files found" "no-videos dir reports no videos"

# --- Multiple video extensions ---

dir="$TEST_DIR/extensions"
place_video "$dir/a.mp4"
cp "$FIXTURE_VIDEO" "$dir/b.mov"
cp "$FIXTURE_VIDEO" "$dir/c.mkv"
cp "$FIXTURE_VIDEO" "$dir/d.avi"

output=$(run_vc "$dir" --strategy balanced --suffix "_test" --dry-run 2>&1) && rc=0 || rc=$?
assert_exit 0 "$rc" "multiple extensions scan exits 0"
assert_output_contains "$output" "Would encode 4 files" "finds .mp4 .mov .mkv .avi"

summarise

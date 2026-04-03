#!/usr/bin/env bash
source "$(dirname "$0")/../lib.sh"
setup_tmp

echo "compress flag combinations (2s clip)"

# --- Strategy variants ---

for strategy in quality balanced size; do
    dir="$TEST_DIR/strategy_$strategy"
    mkdir -p "$dir"
    place_video "$dir/clip.mp4"

    run_vc "$dir" --strategy "$strategy" --suffix "_${strategy}" >/dev/null 2>&1 && rc=0 || rc=$?
    assert_exit 0 "$rc" "compress --strategy $strategy exits 0"
    assert_file_exists "$dir/clip_${strategy}.mp4" "compress --strategy $strategy produces output"
    assert_file_nonzero "$dir/clip_${strategy}.mp4" "compress --strategy $strategy output is non-empty"
done

# --- Resolution variants ---

for res in original 720p 1080p 4k; do
    dir="$TEST_DIR/res_$res"
    mkdir -p "$dir"
    place_video "$dir/clip.mp4"

    run_vc "$dir" --strategy balanced --resolution "$res" --suffix "_${res}" >/dev/null 2>&1 && rc=0 || rc=$?
    assert_exit 0 "$rc" "compress --resolution $res exits 0"
    assert_file_exists "$dir/clip_${res}.mp4" "compress --resolution $res produces output"
done

# --- FPS variants ---

for fps in 0 24 30 60; do
    dir="$TEST_DIR/fps_$fps"
    mkdir -p "$dir"
    place_video "$dir/clip.mp4"

    run_vc "$dir" --strategy balanced --fps "$fps" --suffix "_fps${fps}" >/dev/null 2>&1 && rc=0 || rc=$?
    assert_exit 0 "$rc" "compress --fps $fps exits 0"
    assert_file_exists "$dir/clip_fps${fps}.mp4" "compress --fps $fps produces output"
done

# --- Audio variants ---

for audio in keep low medium high; do
    dir="$TEST_DIR/audio_$audio"
    mkdir -p "$dir"
    place_video "$dir/clip.mp4"

    run_vc "$dir" --strategy balanced --audio "$audio" --suffix "_${audio}" >/dev/null 2>&1 && rc=0 || rc=$?
    assert_exit 0 "$rc" "compress --audio $audio exits 0"
    assert_file_exists "$dir/clip_${audio}.mp4" "compress --audio $audio produces output"
done

# --- Custom suffix ---

dir="$TEST_DIR/suffix_custom"
mkdir -p "$dir"
place_video "$dir/clip.mp4"

run_vc "$dir" --strategy balanced --suffix "_v2" >/dev/null 2>&1 && rc=0 || rc=$?
assert_exit 0 "$rc" "compress --suffix _v2 exits 0"
assert_file_exists "$dir/clip_v2.mp4" "custom suffix produces correct filename"

# --- Workers flag ---

dir="$TEST_DIR/workers"
mkdir -p "$dir"
place_video "$dir/clip.mp4"

run_vc "$dir" --strategy balanced --workers 1 --suffix "_w1" >/dev/null 2>&1 && rc=0 || rc=$?
assert_exit 0 "$rc" "compress --workers 1 exits 0"
assert_file_exists "$dir/clip_w1.mp4" "compress --workers 1 produces output"

# --- Compressed output is smaller than original ---

dir="$TEST_DIR/size_check"
mkdir -p "$dir"
place_video "$dir/clip.mp4"

run_vc "$dir" --strategy size --suffix "_small" >/dev/null 2>&1 && rc=0 || rc=$?
original_size=$(stat -c%s "$dir/clip.mp4" 2>/dev/null || stat -f%z "$dir/clip.mp4" 2>/dev/null)
assert_file_smaller_than "$dir/clip_small.mp4" "$original_size" "size strategy output is smaller than original"

summarise

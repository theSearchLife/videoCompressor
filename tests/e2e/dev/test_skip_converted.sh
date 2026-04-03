#!/usr/bin/env bash
source "$(dirname "$0")/../lib.sh"
setup_tmp

echo "skip-converted behaviour"

# --- First run encodes, second run with --skip-converted yes skips ---

dir="$TEST_DIR/skip"
mkdir -p "$dir"
place_video "$dir/clip.mp4"

# First compress
run_vc "$dir" --strategy balanced --suffix "_test" >/dev/null 2>&1
assert_file_exists "$dir/clip_test.mp4" "first compress produces output"

# Record mtime of output
mtime_before=$(stat -c%Y "$dir/clip_test.mp4" 2>/dev/null || stat -f%m "$dir/clip_test.mp4" 2>/dev/null)
sleep 1

# Second compress with skip-converted yes (default)
output=$(run_vc "$dir" --strategy balanced --suffix "_test" --skip-converted yes 2>&1) && rc=0 || rc=$?
assert_exit 0 "$rc" "skip-converted yes exits 0"

mtime_after=$(stat -c%Y "$dir/clip_test.mp4" 2>/dev/null || stat -f%m "$dir/clip_test.mp4" 2>/dev/null)
_TOTAL=$((_TOTAL + 1))
if [[ "$mtime_before" -eq "$mtime_after" ]]; then
    _PASS=$((_PASS + 1))
    printf "  \033[32mPASS\033[0m skip-converted yes does not re-encode\n"
else
    _FAIL=$((_FAIL + 1))
    printf "  \033[31mFAIL\033[0m skip-converted yes re-encoded (mtime changed)\n"
fi

# --- Re-encode with --skip-converted no ---

output=$(run_vc "$dir" --strategy balanced --suffix "_test" --skip-converted no 2>&1) && rc=0 || rc=$?
assert_exit 0 "$rc" "skip-converted no exits 0"

mtime_reencoded=$(stat -c%Y "$dir/clip_test.mp4" 2>/dev/null || stat -f%m "$dir/clip_test.mp4" 2>/dev/null)
_TOTAL=$((_TOTAL + 1))
if [[ "$mtime_reencoded" -gt "$mtime_before" ]]; then
    _PASS=$((_PASS + 1))
    printf "  \033[32mPASS\033[0m skip-converted no re-encodes the file\n"
else
    _FAIL=$((_FAIL + 1))
    printf "  \033[31mFAIL\033[0m skip-converted no did not re-encode (mtime unchanged)\n"
fi

summarise

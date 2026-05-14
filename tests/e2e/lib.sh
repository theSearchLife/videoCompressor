#!/usr/bin/env bash
# Shared helpers for e2e tests.
# Source this file from test scripts: source "$(dirname "$0")/../lib.sh"

set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
VC_IMAGE="${VC_IMAGE:-vc:runtime}"
VC_DOCKER_USER="${VC_DOCKER_USER:-$(id -u):$(id -g)}"
_VC_IMAGE_CHECKED=0

# Counters
_PASS=0
_FAIL=0
_TOTAL=0

# --- Temp dir management ---

setup_tmp() {
    TEST_DIR="$(mktemp -d)"
    trap 'rm -rf "$TEST_DIR"' EXIT
}

require_vc_image() {
    if [[ "$_VC_IMAGE_CHECKED" -eq 1 ]]; then
        return
    fi
    if ! docker image inspect "$VC_IMAGE" >/dev/null 2>&1; then
        echo "Docker image not found: $VC_IMAGE" >&2
        echo "Build it first with: docker build --target runtime -t $VC_IMAGE $REPO_ROOT" >&2
        exit 1
    fi
    _VC_IMAGE_CHECKED=1
}

ensure_fixture_video() {
    require_vc_image
    if [[ -z "${TEST_DIR:-}" ]]; then
        echo "setup_tmp must be called before place_video" >&2
        exit 1
    fi

    FIXTURE_VIDEO="$TEST_DIR/.fixture/synthetic_2s_1080p_high_bitrate.mp4"
    if [[ -f "$FIXTURE_VIDEO" ]]; then
        return
    fi

    mkdir -p "$(dirname "$FIXTURE_VIDEO")"
    docker run --rm --user "$VC_DOCKER_USER" --entrypoint ffmpeg \
        -v "$(dirname "$FIXTURE_VIDEO"):/fixture" \
        "$VC_IMAGE" \
        -v error -y \
        -f lavfi -i "testsrc2=duration=2:size=1920x1080:rate=30" \
        -f lavfi -i "sine=frequency=440:duration=2" \
        -c:v mpeg4 -b:v 30M -minrate 30M -maxrate 30M -bufsize 60M \
        -c:a aac -b:a 320k -shortest \
        /fixture/synthetic_2s_1080p_high_bitrate.mp4 >/dev/null
}

# Copy the 2s fixture video into a target path (creating parent dirs).
place_video() {
    local dest="$1"
    ensure_fixture_video
    mkdir -p "$(dirname "$dest")"
    cp "$FIXTURE_VIDEO" "$dest"
}

# Create a non-video file (empty) at the given path.
place_file() {
    local dest="$1"
    mkdir -p "$(dirname "$dest")"
    touch "$dest"
}

# --- Assertions ---

assert_exit() {
    local expected="$1" actual="$2" label="$3"
    _TOTAL=$((_TOTAL + 1))
    if [[ "$actual" -eq "$expected" ]]; then
        _PASS=$((_PASS + 1))
        printf "  \033[32mPASS\033[0m %s\n" "$label"
    else
        _FAIL=$((_FAIL + 1))
        printf "  \033[31mFAIL\033[0m %s (expected exit %d, got %d)\n" "$label" "$expected" "$actual"
    fi
}

assert_file_exists() {
    local path="$1" label="$2"
    _TOTAL=$((_TOTAL + 1))
    if [[ -f "$path" ]]; then
        _PASS=$((_PASS + 1))
        printf "  \033[32mPASS\033[0m %s\n" "$label"
    else
        _FAIL=$((_FAIL + 1))
        printf "  \033[31mFAIL\033[0m %s (file not found: %s)\n" "$label" "$path"
    fi
}

assert_file_not_exists() {
    local path="$1" label="$2"
    _TOTAL=$((_TOTAL + 1))
    if [[ ! -f "$path" ]]; then
        _PASS=$((_PASS + 1))
        printf "  \033[32mPASS\033[0m %s\n" "$label"
    else
        _FAIL=$((_FAIL + 1))
        printf "  \033[31mFAIL\033[0m %s (file should not exist: %s)\n" "$label" "$path"
    fi
}

assert_dir_empty() {
    local dir="$1" label="$2"
    _TOTAL=$((_TOTAL + 1))
    local count
    count=$(find "$dir" -maxdepth 1 -type f | wc -l)
    if [[ "$count" -eq 0 ]]; then
        _PASS=$((_PASS + 1))
        printf "  \033[32mPASS\033[0m %s\n" "$label"
    else
        _FAIL=$((_FAIL + 1))
        printf "  \033[31mFAIL\033[0m %s (expected empty dir, found %d files)\n" "$label" "$count"
    fi
}

assert_output_contains() {
    local output="$1" pattern="$2" label="$3"
    _TOTAL=$((_TOTAL + 1))
    if echo "$output" | grep -qE "$pattern"; then
        _PASS=$((_PASS + 1))
        printf "  \033[32mPASS\033[0m %s\n" "$label"
    else
        _FAIL=$((_FAIL + 1))
        printf "  \033[31mFAIL\033[0m %s (output did not match: %s)\n" "$label" "$pattern"
    fi
}

assert_output_not_contains() {
    local output="$1" pattern="$2" label="$3"
    _TOTAL=$((_TOTAL + 1))
    if ! echo "$output" | grep -qE "$pattern"; then
        _PASS=$((_PASS + 1))
        printf "  \033[32mPASS\033[0m %s\n" "$label"
    else
        _FAIL=$((_FAIL + 1))
        printf "  \033[31mFAIL\033[0m %s (output should not match: %s)\n" "$label" "$pattern"
    fi
}

assert_file_smaller_than() {
    local path="$1" max_bytes="$2" label="$3"
    _TOTAL=$((_TOTAL + 1))
    if [[ ! -f "$path" ]]; then
        _FAIL=$((_FAIL + 1))
        printf "  \033[31mFAIL\033[0m %s (file not found: %s)\n" "$label" "$path"
        return
    fi
    local size
    size=$(stat -c%s "$path" 2>/dev/null || stat -f%z "$path" 2>/dev/null)
    if [[ "$size" -lt "$max_bytes" ]]; then
        _PASS=$((_PASS + 1))
        printf "  \033[32mPASS\033[0m %s\n" "$label"
    else
        _FAIL=$((_FAIL + 1))
        printf "  \033[31mFAIL\033[0m %s (file %d bytes, expected < %d)\n" "$label" "$size" "$max_bytes"
    fi
}

assert_file_nonzero() {
    local path="$1" label="$2"
    _TOTAL=$((_TOTAL + 1))
    if [[ ! -f "$path" ]]; then
        _FAIL=$((_FAIL + 1))
        printf "  \033[31mFAIL\033[0m %s (file not found: %s)\n" "$label" "$path"
        return
    fi
    local size
    size=$(stat -c%s "$path" 2>/dev/null || stat -f%z "$path" 2>/dev/null)
    if [[ "$size" -gt 0 ]]; then
        _PASS=$((_PASS + 1))
        printf "  \033[32mPASS\033[0m %s\n" "$label"
    else
        _FAIL=$((_FAIL + 1))
        printf "  \033[31mFAIL\033[0m %s (file is empty)\n" "$label"
    fi
}

# --- Runner ---

# Run vc and capture exit code + combined output. Non-interactive (no TTY).
run_vc() {
    require_vc_image

    local args=("$@")
    local docker_args=(docker run --rm --user "$VC_DOCKER_USER")

    map_existing_dir() {
        local index="$1"
        if [[ "${#args[@]}" -gt "$index" && -d "${args[$index]}" ]]; then
            docker_args+=(-v "${args[$index]}:/videos")
            args[$index]="/videos"
        fi
    }

    case "${args[0]:-}" in
        cleanup|assess|compress)
            map_existing_dir 1
            ;;
        ""|--help|-h|help)
            ;;
        *)
            map_existing_dir 0
            ;;
    esac

    local output rc=0
    output=$("${docker_args[@]}" "$VC_IMAGE" "${args[@]}" 2>&1) || rc=$?
    echo "$output"
    return $rc
}

# Print test file summary and return appropriate exit code.
summarise() {
    local file="${1:-$(basename "$0")}"
    echo ""
    if [[ $_FAIL -eq 0 ]]; then
        printf "\033[32m%s: %d/%d passed\033[0m\n" "$file" "$_PASS" "$_TOTAL"
    else
        printf "\033[31m%s: %d/%d passed (%d failed)\033[0m\n" "$file" "$_PASS" "$_TOTAL" "$_FAIL"
    fi
    return "$_FAIL"
}

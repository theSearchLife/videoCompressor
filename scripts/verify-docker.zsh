#!/usr/bin/env zsh

set -euo pipefail

REPO_ROOT="${0:A:h:h}"
cd "$REPO_ROOT"

TEST_IMAGE="${VC_TEST_IMAGE:-vc:unit-test}"
RUNTIME_IMAGE="${VC_IMAGE:-vc:runtime}"
SLOG3_ROOT="${VC_SLOG3_SAMPLE_ROOT:-/tmp/vc-slog3-samples}"
USER_GROUP="$(id -u):$(id -g)"

log() {
    printf '\n==> %s\n' "$*"
}

run() {
    printf '+'
    printf ' %q' "$@"
    printf '\n'
    "$@"
}

run_in_test_image() {
    run docker run --rm "$TEST_IMAGE" "$@"
}

run_in_runtime_image() {
    run docker run --rm "$RUNTIME_IMAGE" "$@"
}

assert_runtime_contract() {
    run docker run --rm --entrypoint sh "$RUNTIME_IMAGE" -lc '
        set -eu
        command -v vc >/dev/null
        command -v ffmpeg >/dev/null
        command -v ffprobe >/dev/null
        command -v mediainfo >/dev/null
        if command -v go >/dev/null 2>&1; then
            echo "runtime image must not contain the Go toolchain" >&2
            exit 1
        fi
    '
}

run_static_checks() {
    run docker run --rm "$TEST_IMAGE" sh -lc '
        set -eu
        files="$(find . -name "*.go" -type f -print0 | xargs -0 /usr/local/go/bin/gofmt -l)"
        if [ -n "$files" ]; then
            echo "Go files need gofmt:" >&2
            echo "$files" >&2
            exit 1
        fi
    '
    run_in_test_image go vet ./...
    run docker run --rm "$TEST_IMAGE" sh -lc '
        set -eu
        for file in tests/e2e/run.sh tests/e2e/lib.sh tests/e2e/dev/*.sh tests/e2e/post-build/*.sh; do
            bash -n "$file"
        done
        for file in scripts/verify-docker.zsh scripts/verify-delivery.zsh scripts/test-progress.sh; do
            zsh -n "$file"
        done
    '
}

run_slog3_validation() {
    if [ ! -d "$SLOG3_ROOT" ]; then
        log "Skipping S-Log3 validation; sample root not found: $SLOG3_ROOT"
        return
    fi

    local slog_dir="$SLOG3_ROOT/slog3"
    if [ ! -d "$slog_dir" ]; then
        log "Skipping S-Log3 validation; sample directory not found: $slog_dir"
        return
    fi

    log "Running S-Log3 dry-run detection from $slog_dir"
    local slog_output
    slog_output="$(
        docker run --rm \
            -v "$slog_dir:/samples:ro" \
            "$RUNTIME_IMAGE" /samples \
            --strategy balanced \
            --resolution original \
            --fps 0 \
            --audio keep \
            --suffix _verify \
            --skip-converted no \
            --workers 1 \
            --dry-run 2>&1
    )"
    printf '%s\n' "$slog_output"
    if ! printf '%s\n' "$slog_output" | grep -q 'S-Log3 detected'; then
        echo "expected S-Log3 detection log for client samples" >&2
        exit 1
    fi

    local normal_sample="$SLOG3_ROOT/normal_1s.mp4"
    if [ ! -f "$normal_sample" ]; then
        log "Skipping normal-sample false-positive check; not found: $normal_sample"
        return
    fi

	local normal_dir
	normal_dir="$(mktemp -d)"
	trap "rm -rf ${(q)normal_dir}" EXIT
    ln "$normal_sample" "$normal_dir/normal_1s.mp4" || cp "$normal_sample" "$normal_dir/normal_1s.mp4"

    log "Running non-S-Log3 dry-run false-positive check"
    local normal_output
    normal_output="$(
        docker run --rm \
            -v "$normal_dir:/samples:ro" \
            "$RUNTIME_IMAGE" /samples \
            --strategy balanced \
            --resolution original \
            --fps 0 \
            --audio keep \
            --suffix _verify \
            --skip-converted no \
            --workers 1 \
            --dry-run 2>&1
    )"
    printf '%s\n' "$normal_output"
    if printf '%s\n' "$normal_output" | grep -q 'S-Log3 detected'; then
        echo "normal sample was incorrectly detected as S-Log3" >&2
        exit 1
    fi
}

log "Building project test image: $TEST_IMAGE"
run docker build --target unit-test -t "$TEST_IMAGE" .

log "Running unit tests inside project test image"
run_in_test_image go test ./...

log "Running static checks inside project test image"
run_static_checks

log "Building runtime image: $RUNTIME_IMAGE"
run docker build --target runtime -t "$RUNTIME_IMAGE" .

log "Checking runtime image contract"
assert_runtime_contract

log "Running runtime help smoke test"
run_in_runtime_image --help

log "Running e2e suites through runtime image"
VC_IMAGE="$RUNTIME_IMAGE" VC_DOCKER_USER="$USER_GROUP" tests/e2e/run.sh all

run_slog3_validation

log "Docker verification complete"

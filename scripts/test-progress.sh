#!/usr/bin/env zsh
# Smoke-test the live multi-line progress display.
#
# Usage:
#   scripts/test-progress.sh                # synthesise 4 sample videos
#   scripts/test-progress.sh /path/to/dir   # use your own folder
#
# Notes:
# - Builds the local docker image as vc:test if it doesn't already exist.
# - Runs with -t so stdout is a TTY (the live region is suppressed otherwise).
# - Bump WORKERS=N to see more concurrent live lines.

set -euo pipefail

IMAGE="${IMAGE:-vc:test}"
WORKERS="${WORKERS:-2}"
SAMPLES="${SAMPLES:-4}"
SAMPLE_SECS="${SAMPLE_SECS:-30}"

REPO_ROOT="$(cd "$(dirname "$0")/.." && pwd)"

if ! docker image inspect "$IMAGE" >/dev/null 2>&1; then
    echo "Building $IMAGE from $REPO_ROOT..."
    docker build -t "$IMAGE" "$REPO_ROOT"
fi

if [[ $# -ge 1 ]]; then
    VIDEO_DIR="$1"
    CLEANUP=false
else
    VIDEO_DIR="$(mktemp -d)"
    CLEANUP=true
    echo "Generating $SAMPLES synthetic ${SAMPLE_SECS}s samples in $VIDEO_DIR..."
    for i in $(seq 1 "$SAMPLES"); do
        docker run --rm --entrypoint ffmpeg -v "$VIDEO_DIR:/v" "$IMAGE" \
            -y \
            -f lavfi -i "testsrc=duration=${SAMPLE_SECS}:size=1920x1080:rate=30" \
            -f lavfi -i "sine=frequency=440:duration=${SAMPLE_SECS}" \
            -c:v mpeg4 -q:v 1 -c:a pcm_s16le -shortest \
            "/v/sample${i}.avi" >/dev/null 2>&1
    done
fi

cleanup() {
    if [[ "$CLEANUP" == true ]]; then
        rm -rf "$VIDEO_DIR"
    fi
}
trap cleanup EXIT

echo "Running compress against $VIDEO_DIR with $WORKERS workers..."
echo

docker run --rm -t -v "$VIDEO_DIR:/videos" "$IMAGE" /videos \
    --strategy balanced \
    --resolution original \
    --fps 0 \
    --audio keep \
    --suffix _compressed \
    --skip-converted no \
    --workers "$WORKERS"

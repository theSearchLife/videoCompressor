#!/usr/bin/env bash
# E2E test runner.
# Usage: ./run.sh dev          Run development tests (comprehensive)
#        ./run.sh post-build   Run post-build smoke tests
#        ./run.sh              Run both suites
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"

suite="${1:-all}"
overall_fail=0

run_suite() {
    local dir="$1" label="$2"
    echo ""
    echo "========================================"
    echo " $label"
    echo "========================================"

    local test_files=()
    while IFS= read -r f; do
        test_files+=("$f")
    done < <(find "$dir" -name 'test_*.sh' -type f | sort)

    if [[ ${#test_files[@]} -eq 0 ]]; then
        echo "No tests found in $dir"
        return
    fi

    for test_file in "${test_files[@]}"; do
        echo ""
        echo "--- $(basename "$test_file") ---"
        if ! bash "$test_file"; then
            overall_fail=1
        fi
    done
}

case "$suite" in
    dev)
        run_suite "$SCRIPT_DIR/dev" "Dev Tests"
        ;;
    post-build)
        run_suite "$SCRIPT_DIR/post-build" "Post-Build Smoke Tests"
        ;;
    all)
        run_suite "$SCRIPT_DIR/dev" "Dev Tests"
        run_suite "$SCRIPT_DIR/post-build" "Post-Build Smoke Tests"
        ;;
    *)
        echo "Usage: $0 [dev|post-build|all]" >&2
        exit 1
        ;;
esac

echo ""
if [[ $overall_fail -eq 0 ]]; then
    printf "\033[32mAll suites passed.\033[0m\n"
else
    printf "\033[31mSome tests failed.\033[0m\n"
    exit 1
fi

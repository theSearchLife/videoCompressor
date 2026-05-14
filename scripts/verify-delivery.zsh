#!/usr/bin/env zsh

set -euo pipefail

REPO_ROOT="${0:A:h:h}"
cd "$REPO_ROOT"

RUNTIME_IMAGE="${VC_IMAGE:-vc:runtime}"
SAMPLE_ROOT="${VC_SLOG3_SAMPLE_ROOT:-/tmp/vc-slog3-samples}"
KEEP_OUTPUTS="${VC_DELIVERY_KEEP_OUTPUTS:-0}"
USER_GROUP="${VC_DOCKER_USER:-$(id -u):$(id -g)}"

WORKDIR=""

log() {
    printf '\n==> %s\n' "$*"
}

run() {
    printf '+'
    printf ' %q' "$@"
    printf '\n'
    "$@"
}

die() {
    printf 'ERROR: %s\n' "$*" >&2
    exit 1
}

usage() {
    cat <<'EOF'
Usage: scripts/verify-delivery.zsh

Environment:
  VC_IMAGE                    Runtime image tag (default: vc:runtime)
  VC_SLOG3_SAMPLE_ROOT        Sample root (default: /tmp/vc-slog3-samples)
  VC_DELIVERY_SLOG3_SAMPLE    Specific S-Log3 sample path or name under slog3/
  VC_DELIVERY_NORMAL_SAMPLE   Specific normal sample path or name under the sample root
  VC_DELIVERY_KEEP_OUTPUTS=1  Preserve the temporary workdir
EOF
}

cleanup() {
    local rc=$?
    if [[ -n "$WORKDIR" ]]; then
        if [[ "$KEEP_OUTPUTS" == "1" ]]; then
            log "Preserved delivery validation workdir: $WORKDIR"
        else
            rm -rf "$WORKDIR"
        fi
    fi
    exit "$rc"
}
trap cleanup EXIT

case "${1:-}" in
    "")
        ;;
    -h|--help)
        usage
        exit 0
        ;;
    *)
        die "unexpected argument: $1"
        ;;
esac

supported_video_file() {
    local file_path="$1"
    case "${file_path:e:l}" in
        mp4|mkv|avi|mov|wmv|flv|webm|m4v|mpg|mpeg|3gp|ts)
            return 0
            ;;
        *)
            return 1
            ;;
    esac
}

stat_size() {
    local file_path="$1"
    stat -c%s "$file_path" 2>/dev/null || stat -f%z "$file_path"
}

resolve_sample_path() {
    local configured="$1"
    local fallback_dir="$2"
    local label="$3"
    local candidate="$configured"

    if [[ "$candidate" != /* && -f "$fallback_dir/$candidate" ]]; then
        candidate="$fallback_dir/$candidate"
    fi

    [[ -f "$candidate" ]] || die "$label sample not found: $configured"
    supported_video_file "$candidate" || die "$label sample is not a recognised video type: $candidate"
    printf '%s\n' "${candidate:A}"
}

select_smallest_slog3_sample() {
    local slog_dir="$1"
    local smallest=""
    local smallest_size=0
    local file size

    while IFS= read -r -d '' file; do
        supported_video_file "$file" || continue
        size="$(stat_size "$file")"
        if [[ -z "$smallest" || "$size" -lt "$smallest_size" ]]; then
            smallest="$file"
            smallest_size="$size"
        fi
    done < <(find "$slog_dir" -type f -print0)

    [[ -n "$smallest" ]] || die "no recognised S-Log3 video samples found in $slog_dir"
    printf '%s\n' "${smallest:A}"
}

hardlink_sample() {
    local source="$1"
    local dest_dir="$2"
    local dest="$dest_dir/${source:t}"

    mkdir -p "$dest_dir"
    if ! ln "$source" "$dest" 2>/dev/null; then
        die "failed to hardlink $source into $dest_dir. Delivery validation refuses to copy large samples; put the temp workdir and samples on the same filesystem or set VC_SLOG3_SAMPLE_ROOT accordingly."
    fi

    printf '%s\n' "$dest"
}

output_for() {
    local input="$1"
    local suffix="$2"
    local dir="${input:h}"
    local base="${input:t}"
    local stem="${base%.*}"
    printf '%s/%s%s.mp4\n' "$dir" "$stem" "$suffix"
}

docker_vc() {
    local input_dir="$1"
    shift
    docker run --rm \
        --user "$USER_GROUP" \
        -v "$input_dir:/videos" \
        "$RUNTIME_IMAGE" /videos "$@"
}

probe_stream_field() {
    local output="$1"
    local field="$2"
    local dir="${output:h}"
    local file="${output:t}"

    docker run --rm \
        --user "$USER_GROUP" \
        --entrypoint ffprobe \
        -v "$dir:/probe:ro" \
        "$RUNTIME_IMAGE" \
        -v error \
        -select_streams v:0 \
        -show_entries "stream=$field" \
        -of default=noprint_wrappers=1:nokey=1 \
        "/probe/$file"
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

run_delivery_case() {
    local case_name="$1"
    local sample_kind="$2"
    local source_sample="$3"
    local strategy="$4"
    local resolution="$5"
    local fps="$6"
    local audio="$7"
    local suffix="_delivery_${case_name}_${sample_kind}"
    local case_dir="$WORKDIR/$case_name/$sample_kind"
    local input output log_path command_output rc

    input="$(hardlink_sample "$source_sample" "$case_dir")"
    output="$(output_for "$input" "$suffix")"
    log_path="$case_dir/vc.log"

    log "Case $case_name / $sample_kind: strategy=$strategy resolution=$resolution fps=$fps audio=$audio"
    set +e
    command_output="$(docker_vc "$case_dir" \
        --strategy "$strategy" \
        --resolution "$resolution" \
        --fps "$fps" \
        --audio "$audio" \
        --suffix "$suffix" \
        --skip-converted no \
        --workers 1 2>&1)"
    rc=$?
    set -e
    printf '%s\n' "$command_output" > "$log_path"

    if [[ "$rc" -ne 0 ]]; then
        printf '%s\n' "$command_output" >&2
        die "$sample_kind compression failed for $case_name with exit $rc"
    fi

    [[ -f "$output" ]] || die "$sample_kind output missing for $case_name: $output"
    [[ -s "$output" ]] || die "$sample_kind output is empty for $case_name: $output"

    if [[ "$sample_kind" == "slog3" ]]; then
        if ! printf '%s\n' "$command_output" | grep -q 'S-Log3 detected'; then
            printf '%s\n' "$command_output" >&2
            die "S-Log3 detection log missing for $case_name"
        fi

        local codec pix_fmt
        codec="$(probe_stream_field "$output" codec_name)"
        pix_fmt="$(probe_stream_field "$output" pix_fmt)"
        [[ "$codec" == "hevc" ]] || die "S-Log3 output codec for $case_name is $codec, expected hevc"
        [[ "$pix_fmt" == "yuv420p10le" ]] || die "S-Log3 output pix_fmt for $case_name is $pix_fmt, expected yuv420p10le"
    else
        if printf '%s\n' "$command_output" | grep -q 'S-Log3 detected'; then
            printf '%s\n' "$command_output" >&2
            die "normal sample was incorrectly detected as S-Log3 for $case_name"
        fi
    fi
}

[[ -d "$SAMPLE_ROOT" ]] || die "S-Log3 sample root not found: $SAMPLE_ROOT. Set VC_SLOG3_SAMPLE_ROOT to the directory containing slog3/ and normal_1s.mp4."

slog_dir="$SAMPLE_ROOT/slog3"
[[ -d "$slog_dir" ]] || die "S-Log3 sample directory not found: $slog_dir"

if [[ -n "${VC_DELIVERY_SLOG3_SAMPLE:-}" ]]; then
    slog_sample="$(resolve_sample_path "$VC_DELIVERY_SLOG3_SAMPLE" "$slog_dir" "S-Log3")"
else
    slog_sample="$(select_smallest_slog3_sample "$slog_dir")"
fi

if [[ -n "${VC_DELIVERY_NORMAL_SAMPLE:-}" ]]; then
    normal_sample="$(resolve_sample_path "$VC_DELIVERY_NORMAL_SAMPLE" "$SAMPLE_ROOT" "normal")"
else
    normal_sample="$(resolve_sample_path "$SAMPLE_ROOT/normal_1s.mp4" "$SAMPLE_ROOT" "normal")"
fi

log "Building runtime image: $RUNTIME_IMAGE"
run docker build --target runtime -t "$RUNTIME_IMAGE" .

log "Checking runtime image contract"
assert_runtime_contract

if ! WORKDIR="$(mktemp -d "$SAMPLE_ROOT/.vc-delivery.XXXXXXXX")"; then
    die "failed to create delivery validation workdir under $SAMPLE_ROOT"
fi
log "Using S-Log3 sample: $slog_sample"
log "Using normal sample: $normal_sample"
log "Delivery validation workdir: $WORKDIR"

# This is axis coverage, not Cartesian coverage. Companion settings are chosen
# to force a real encode for the efficient normal_1s.mp4 sample instead of
# hitting compression-advice skips at original/keep settings.
cases=(
    'strategy_quality|quality|720p|24|low'
    'strategy_balanced|balanced|720p|24|low'
    'strategy_size|size|720p|24|low'
    'resolution_original|size|original|24|low'
    'resolution_720p|size|720p|24|low'
    'resolution_1080p|size|1080p|24|low'
    'fps_0|size|720p|0|low'
    'fps_24|size|720p|24|low'
    'fps_30|size|720p|30|low'
    'audio_keep|size|720p|24|keep'
    'audio_low|size|720p|24|low'
    'audio_medium|size|720p|24|medium'
    'audio_high|size|720p|24|high'
)

for case_spec in "${cases[@]}"; do
    IFS='|' read -r case_name strategy resolution fps audio <<< "$case_spec"
    run_delivery_case "$case_name" "slog3" "$slog_sample" "$strategy" "$resolution" "$fps" "$audio"
    run_delivery_case "$case_name" "normal" "$normal_sample" "$strategy" "$resolution" "$fps" "$audio"
done

log "Delivery validation complete"

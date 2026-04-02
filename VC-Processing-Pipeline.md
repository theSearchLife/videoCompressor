# VC Processing Pipeline

## Pipeline

```text
Scan -> Probe -> Plan -> Encode -> Report
```

## Scan

- Walk the input directory recursively.
- Ignore non-video files.
- Ignore empty video files.
- Delete stale `*.tmp` files before returning the candidate video list.
- Sort candidates by size descending.

## Probe

- Use `ffprobe` JSON output.
- Capture width, height, duration, codec, and file size.

## Plan

- Resolve effective output resolution without upscaling.
- Build the compression profile.
- Compute the final output path.
- Compute the temporary output path as `<final>.tmp`.
- Ignore previously converted suffixed outputs as source inputs.
- Remove a stale matching `<final>.tmp` path before scheduling the next encode.
- Skip work when the expected final output already exists.

## Encode

- Use `ffmpeg`.
- Write to the temporary output path.
- Parse progress from `-progress pipe:1`.
- On success, atomically rename the temporary path to the final path.
- On failure, leave the temporary file for next-run cleanup.

## Report

- Emit per-file progress updates during execution.
- Emit a batch summary after completion.
- Assessment runs also write `report.md` and `results.csv`.

## Cleanup

- Re-scan the input tree recursively.
- Probe original candidates to resolve the effective cleanup suffix.
- Delete the original file when the matching converted suffixed `.mp4` exists.
- Atomically rename the converted suffixed `.mp4` to the unsuffixed `.mp4` path.

## ffmpeg Rules

- video codec is driven by the selected profile
- scaling uses `scale=-2:<height>` when downscaling is required
- audio copies when possible and re-encodes when required by profile
- `+faststart` is enabled for `.mp4` outputs

# VC Technical Architecture

## Architecture

The application uses a ports-and-adapters structure.

- `internal/domain` contains pure types and naming rules.
- `internal/port` defines scanner, prober, encoder, and reporter interfaces.
- `internal/adapter/fs` implements recursive file discovery.
- `internal/adapter/ffmpeg` implements probing, encoding, and progress parsing.
- `internal/app` coordinates batch execution and assessment runs.
- `cmd/vc` wires the runtime and CLI entrypoints.

## Runtime Decisions

- Runtime delivery is Docker-first.
- Linux, macOS, and Windows all execute through Docker.
- The runtime image contains the Go binary plus `ffmpeg` and `ffprobe`.

## Commands

- `vc compress <input-dir>` runs batch compression.
- `vc cleanup <input-dir> --resolution <res>` runs the second-pass original replacement flow.
- `vc assess <input-dir> --output <dir>` runs the internal comparison matrix.

## Compression Decisions

- Input scanning is always recursive.
- Mixed-content directories are supported.
- Only recognised video files are included.
- Final compression outputs are `.mp4`.
- Temporary outputs are `<final>.tmp`.
- Encode success performs an atomic rename from temporary path to final path.
- Stale `*.tmp` files are deleted during scan before planning work.
- Planning skips completed conversions and ignores converted outputs as fresh inputs.
- Cleanup deletes the original file only after the matching converted output is present, then renames the converted `.mp4` into place.

## Concurrency

- Compression uses a worker pool.
- Default worker count is `CPU/2`.
- Assessment defaults to one worker for stable timing comparisons.

## Interface Decisions

- Compression settings are resolved from flags first, then prompts, then defaults.
- Recursive scanning is not user-configurable.
- Cleanup resolution is resolved from the flag first, then a prompt, then the default.
- Assessment output goes to a separate report directory.

## Docker Examples

### Unix-like hosts

```bash
docker run -it --rm -v /path/to/videos:/videos vc /videos
```

### Windows hosts

```bash
docker run -it --rm -v "C:\Videos:/videos" vc /videos
```

## Project Layout

```text
cmd/vc
internal/domain
internal/port
internal/adapter/fs
internal/adapter/ffmpeg
internal/app
internal/report
testdata/
```

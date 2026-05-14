# VC Technical Architecture

## Architecture

The application uses a ports-and-adapters structure.

- `internal/domain` contains pure types and naming rules.
- `internal/port` defines scanner, prober, encoder, and reporter interfaces.
- `internal/adapter/fs` implements recursive file discovery.
- `internal/adapter/ffmpeg` implements ffprobe/MediaInfo probing, encoding, and progress parsing.
- `internal/app` coordinates batch execution and assessment runs.
- `cmd/vc` wires the runtime and CLI entrypoints.

## Runtime Decisions

- Runtime delivery is Docker-first.
- Linux, macOS, and Windows all execute through Docker.
- The runtime image contains the Go binary plus `ffmpeg`, `ffprobe`, and `mediainfo`.
- The runtime image does not contain the Go toolchain.

## Development Workflow

- `just verify` is the canonical local verification command.
- Verification is Docker-only.
- The Dockerfile exposes a `unit-test` target for Go tests and static checks.
- The Dockerfile exposes a `runtime` target for the client image.
- E2E tests execute `vc` through the runtime image, not through a host binary.
- Optional S-Log3 sample validation reads `/tmp/vc-slog3-samples` through a read-only bind mount.
- `just verify-delivery` runs the slow real-media delivery gate through `scripts/verify-delivery.zsh`.
- Delivery validation builds `vc:runtime`, hardlinks selected S-Log3 and normal samples into a temporary workdir, and refuses to copy large sample media.
- Delivery validation runs option-axis coverage across strategy, resolution, fps, and audio settings. Each case compresses one S-Log3 sample and one normal sample through the runtime image.
- S-Log3 delivery assertions use runtime-image `ffprobe` to require HEVC 10-bit `yuv420p10le` output and the `S-Log3 detected` planning log.

## Commands

- `vc compress <input-dir>` runs batch compression.
- `vc cleanup <input-dir> --resolution <res>` runs the second-pass original replacement flow.
- `vc assess <input-dir> --output <dir>` runs the internal comparison matrix.
- `just verify-delivery` runs real-media S-Log3 delivery validation when local client samples are available.

## Compression Decisions

- Input scanning is always recursive.
- Mixed-content directories are supported.
- Only recognised video files are included.
- Final compression outputs are `.mp4`.
- Temporary outputs are `<final>.tmp`.
- Encode success performs an atomic rename from temporary path to final path.
- Stale `*.tmp` files are deleted during scan before planning work.
- Planning skips completed conversions and ignores converted outputs as fresh inputs.
- Planning marks S-Log3/S-Log3.Cine sources from ffprobe/MediaInfo metadata so the encoder can keep 10-bit H.265 output.
- Cleanup deletes the original file only after the matching converted output is present, then renames the converted `.mp4` into place.

## Concurrency

- Compression uses a worker pool.
- Default worker count is `runtime.NumCPU()` and is exposed in the interactive compression prompts.
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

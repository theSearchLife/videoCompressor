# Project Agent Guide

This repository is a Docker-delivered Go video compressor for the Upwork
Stories / videoCompressor project.

## Status

The Docker-only build, test, and verification path is now the project baseline.
Do not ship or continue feature work after changes unless the canonical verifier
passes.

Current feature work in the working tree includes S-Log3 detection and 10-bit
H.265 handling. Treat it as ready for review only when `just verify` passes from
the current working tree.

## Non-Negotiable Workflow Rules

- Treat this as a Docker-only project.
- Do not run Go tests on the host.
- Do not run `ffmpeg`, `ffprobe`, or `mediainfo` on the host for validation.
- Do not install host media or language dependencies to validate behaviour.
- Do not invent temporary validation harnesses.
- Do not unzip or duplicate large client sample files unless explicitly asked.
- Host commands are acceptable for source inspection, git inspection, and file
  discovery. Host shell scripts may orchestrate Docker and prepare temporary
  test directories, but validation execution must happen inside Docker.

## Canonical Project Surfaces

- Product README: `README.md`
- Behaviour spec: `SPEC.md`
- Technical architecture: `VC-Technical-Architecture.md`
- Docker runtime definition: `Dockerfile`
- Task shortcuts: `justfile`
- Existing e2e runner: `tests/e2e/run.sh`
- Shared e2e helpers: `tests/e2e/lib.sh`
- Real-media delivery gate: `scripts/verify-delivery.zsh`
- CLI entrypoint: `cmd/vc/main.go`
- Domain logic: `internal/domain/`
- App orchestration: `internal/app/`
- FFmpeg/MediaInfo adapter: `internal/adapter/ffmpeg/`

## Required SDLC Target

The repository must expose one deterministic verification command:

```zsh
just verify
```

`just verify` must be Docker-only and must:

1. Build a project-defined test image.
2. Run unit tests inside that test image.
3. Run static checks inside that test image.
4. Build the final runtime image.
5. Assert the runtime image contains `vc`, `ffmpeg`, `ffprobe`, and `mediainfo`.
6. Run runtime smoke tests through the final image.
7. Run e2e tests through the final image.
8. If `/tmp/vc-slog3-samples` exists, run S-Log3 sample validation from that
   directory without extracting or copying the samples.

If `just verify` is broken, the change is not ready to ship.

## Real-Media Delivery Gate

The slow delivery gate is:

```zsh
just verify-delivery
```

It is Docker-only and uses the runtime image `vc:runtime`. It validates real
S-Log3 delivery behaviour against samples in `/tmp/vc-slog3-samples` by default,
or `VC_SLOG3_SAMPLE_ROOT` when set. It selects the smallest recognised video in
`slog3/` unless `VC_DELIVERY_SLOG3_SAMPLE` is set, and uses
`normal_1s.mp4` unless `VC_DELIVERY_NORMAL_SAMPLE` is set.

The gate hardlinks selected samples into a temporary workdir and refuses to copy
large media if hardlinks fail. It runs option-axis coverage across strategy,
resolution, fps, and audio settings for both S-Log3 and normal samples. S-Log3
outputs must be HEVC 10-bit `yuv420p10le` and log `S-Log3 detected`; normal
outputs must not log that detection. Set `VC_DELIVERY_KEEP_OUTPUTS=1` to
preserve the temp workdir for inspection.

## Docker Design Direction

The Dockerfile should provide explicit targets:

- `unit-test`: Go toolchain plus project test dependencies.
- `runtime`: final client image with only the compiled `vc` binary and runtime
  media tools.

The runtime image should not contain the Go toolchain.

## Sample Handling

Known S-Log3 client samples, when present locally, live at:

```text
/tmp/vc-slog3-samples/slog3/
```

The verification workflow may read them through a read-only bind mount. It must
not unzip, move, or duplicate them.

## Historical Workflow Gap

The previous `justfile` mixed project-image commands with generic language
containers. The previous e2e helper also ran the host `./vc` binary directly.
That split was the root workflow defect. Keep all validation on
project-defined Docker targets and Docker-backed e2e execution.

## Implementation Baseline

- `scripts/verify-docker.zsh` is the canonical verifier.
- `just verify` runs the canonical verifier.
- `Dockerfile` exposes `unit-test` and `runtime` targets.
- E2E execution goes through the runtime image.
- Optional S-Log3 sample validation uses the runtime image and reads existing
  samples from `/tmp/vc-slog3-samples`.

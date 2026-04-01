# VC Technical Architecture

Implementation architecture for the Video Compressor CLI tool.

> **Status:** Draft — Pending client sign-off on Phase 0 results.

---

## Overview

Video Compressor (`vc`) is a Go CLI tool that batch-compresses video files using ffmpeg. It runs natively on Linux and inside Docker on Mac/Linux, presenting an interactive TUI for configuration and real-time progress feedback.

### Core Value Proposition

One command to compress an entire directory of videos with optimal quality-to-size ratio. The user picks resolution, compression level, and whether to recurse into subfolders. Everything else — codec selection, CRF tuning, container format, thread allocation — is handled automatically.

---

## Two-Track Design

The tool has two operational modes that share the same hex architecture and ffmpeg adapter:

### Track 1: Assessment (`vc assess`)

Runs a codec/CRF/preset test matrix against sample videos, producing encoded variants with structured filenames and a comparison report. The client can run this themselves to evaluate quality trade-offs.

```
vc assess /videos/samples/ --output /videos/comparison_reports/
```

- Iterates over all matrix combinations (codec × CRF × preset × resolution)
- Output filenames encode the configuration: `{source}_{codec}_crf{N}_{preset}_{resolution}.mp4`
- Generates a timestamped markdown report in `comparison_reports/`
- Encoded test files are temporary — can be deleted after review

### Track 2: Compress (`vc compress` / default)

Normal batch compression using a locked-in profile (decided after assessment).

```
vc compress /videos/
vc /videos/  # compress is the default subcommand
```

- Uses configured profile (from flags, prompts, or defaults)
- Output lands next to source files: `clip.mov` → `clip_720p.mp4`
- Never modifies source files

### Shared Infrastructure

Both tracks use the same:
- `port.Encoder` / `adapter/ffmpeg` — command building, progress parsing
- `port.Scanner` / `adapter/fs` — file discovery
- `port.Prober` / `adapter/ffmpeg` — metadata extraction
- `domain.Profile` — encoding parameters
- Docker image — identical ffmpeg environment

---

## Architecture: Hexagonal (Ports & Adapters)

The system uses hexagonal architecture to isolate the ffmpeg dependency behind a port. This means the core domain logic (file discovery, job orchestration, naming conventions) is testable without ffmpeg, and the encoding backend could be swapped (e.g. HandBrake, hardware encoders) without touching business logic.

```
┌─────────────────────────────────────────────────────┐
│                    CLI / TUI Layer                   │
│         (Bubbletea + Huh + Bubbles + Lipgloss)      │
├─────────────────────────────────────────────────────┤
│                   Application Layer                  │
│            (Job Orchestrator / Coordinator)          │
├──────────┬──────────────────────────────┬───────────┤
│  Port:   │       Port:                  │  Port:    │
│  Scanner │       Encoder                │  Reporter │
├──────────┼──────────────────────────────┼───────────┤
│ Adapter: │  Adapter:                    │ Adapter:  │
│ OS/Walk  │  ffmpeg CLI                  │ TUI/Log   │
└──────────┴──────────────────────────────┴───────────┘
```

### Layer Responsibilities

| Layer | Package | Responsibility |
|-------|---------|----------------|
| **Domain** | `internal/domain` | Types: `Job`, `Profile`, `Resolution`, `CompressionLevel`, `VideoMeta` |
| **Ports** | `internal/port` | Interfaces: `Scanner`, `Encoder`, `Prober`, `Reporter` |
| **Adapters** | `internal/adapter/ffmpeg` | ffmpeg/ffprobe CLI wrapper — builds commands, parses progress |
| **Adapters** | `internal/adapter/fs` | Directory walker, file discovery, glob matching |
| **Application** | `internal/app` | Job orchestrator — fan-out to goroutines, collect results |
| **TUI** | `internal/tui` | Bubbletea models — config form, progress dashboard, summary |
| **CLI** | `cmd/vc` | Entrypoint — parse flags, wire dependencies, launch TUI |

---

## Domain Model

### Core Types

```go
type Resolution string
const (
    Res720p  Resolution = "720p"
    Res1080p Resolution = "1080p"
    Res4K    Resolution = "4k"
)

type CompressionLevel string
const (
    High CompressionLevel = "high" // slow preset, lower CRF
    Low  CompressionLevel = "low"  // fast preset, higher CRF
)

type Profile struct {
    Resolution       Resolution
    Compression      CompressionLevel
    Codec            string // "libx265" or "libx264"
    CRF              int
    Preset           string // "slow" or "fast"
    AudioCodec       string // "aac" or "copy"
    ContainerFormat  string // "mp4"
}

type VideoMeta struct {
    Path       string
    Width      int
    Height     int
    Duration   time.Duration
    Codec      string
    Size       int64
}

type Job struct {
    Input    VideoMeta
    Output   string      // computed output path with suffix
    Profile  Profile
    Status   JobStatus   // pending | encoding | done | failed
    Progress float64     // 0.0 – 1.0
    Error    error
}
```

### Resolution Logic (No Upscale)

The effective resolution is `min(source_height, target_height)`. If a 720p video is processed with a 1080p target, it stays at 720p. The suffix always reflects the *actual* output resolution, not the requested one.

```
source=1080p, target=720p  → scale to 720p,  suffix="_720p"
source=720p,  target=1080p → keep 720p,      suffix="_720p"
source=4k,    target=1080p → scale to 1080p, suffix="_1080p"
```

### Output Naming

#### Compress Mode (Normal)

```
input:  /videos/holiday/clip.mov
output: /videos/holiday/clip_720p.mp4
```

Rules:
- Strip original extension, apply `.mp4`
- Append `_{resolution}` suffix before extension
- Output lives next to the source file (same directory)
- Preserve original file (never overwrite or modify source)
- Skip if output file already exists (idempotent re-runs)

#### Assess Mode (Test Matrix)

```
input:  /videos/samples/LG-Daylight-4K-5s.mp4
output: comparison_reports/2026-03-31T14-30-00/encoded/LG-Daylight-4K-5s_h265_crf26_slow_1080p.mp4
```

Structured filenames encode the full configuration so the client can identify each variant at a glance. All outputs go to a timestamped subdirectory — these are temporary test artefacts.

---

## Technology Stack

| Component | Choice | Rationale |
|-----------|--------|-----------|
| Language | Go 1.24 | Client preference, excellent concurrency, single binary |
| TUI Framework | Bubbletea v2 | Community standard (29k stars), Elm architecture, perfect for concurrent progress |
| Forms/Prompts | Huh v2 | Built on Bubbletea, native Select/Confirm components |
| Progress Bars | Bubbles v2 | Composable progress component, integrates with Bubbletea message loop |
| Styling | Lipgloss v2 | Consistent terminal styling, colour adaptation |
| Video Backend | ffmpeg 7.x | Industry standard, bundled in Docker image |
| Probe | ffprobe | Shipped with ffmpeg, JSON output for metadata extraction |
| Container | Docker | Bundles ffmpeg + Go binary, consistent across Mac/Linux |
| Build | Multi-stage Dockerfile | Go builder → Alpine runtime with ffmpeg |
| Task Runner | just | Single canonical way to build/test/run |

### Import Paths (Charm v2 Vanity Domains)

```go
import (
    tea   "charm.land/bubbletea/v2"
    huh   "charm.land/huh/v2"
    "charm.land/bubbles/v2/progress"
    "charm.land/lipgloss/v2"
)
```

---

## Configuration

### Profile System

Profiles map user-facing choices to ffmpeg parameters. Two built-in profiles; values locked after Phase 0.

```go
type Profile struct {
    Name             string         // "high", "low" (or custom for assess matrix)
    Resolution       Resolution
    Codec            string
    CRF              int
    Preset           string
    AudioCodec       string
    AudioBitrate     string
    ContainerFormat  string
}
```

Built-in profiles (defaults — tuned after Phase 0):

| Profile | Codec | CRF | Preset | Audio | Description |
|---------|-------|-----|--------|-------|-------------|
| `high` | libx265 | 26 | slow | copy/aac 128k | Maximum compression, slower |
| `low` | libx265 | 28 | fast | copy/aac 128k | Quick encode, still good compression |

### Config Precedence

```
CLI flags > interactive prompts > defaults
```

No config file — the tool is deliberately simple. The assess mode generates its own dynamic profiles from the test matrix.

### Key Defaults

| Setting | Default | Override |
|---------|---------|---------|
| Profile | (prompt) | `--compression high\|low` |
| Resolution | (prompt) | `--resolution 720p\|1080p\|4k` |
| Recursive | (prompt) | `--recursive` |
| Workers | CPU/2 | `--workers N` |
| Log output | stderr | `--log-dir ./logs/` |
| Output location | Next to source | (always — never modifies source) |
| Assess output | `./comparison_reports/` | `--output DIR` |

---

## Project Layout

```
videocompressor/
├── cmd/
│   └── vc/
│       └── main.go              # Entrypoint: flag parsing, DI wiring, subcommand dispatch
├── internal/
│   ├── domain/
│   │   ├── types.go             # Resolution, CompressionLevel, Profile, Job, VideoMeta
│   │   ├── profile.go           # Profile builder + built-in profiles
│   │   ├── naming.go            # Output path computation, suffix logic
│   │   └── matrix.go            # Test matrix generation for assess mode
│   ├── port/
│   │   ├── encoder.go           # Encoder interface (Encode, Cancel)
│   │   ├── scanner.go           # Scanner interface (Scan directory → []VideoMeta)
│   │   ├── prober.go            # Prober interface (Probe file → VideoMeta)
│   │   └── reporter.go          # Reporter interface (progress callbacks)
│   ├── adapter/
│   │   ├── ffmpeg/
│   │   │   ├── encoder.go       # ffmpeg command builder + executor
│   │   │   ├── prober.go        # ffprobe JSON parser
│   │   │   └── progress.go      # progress line parser
│   │   └── fs/
│   │       └── scanner.go       # filepath.WalkDir with extension filter
│   ├── app/
│   │   ├── orchestrator.go      # Fan-out encoding jobs, goroutine pool, result collection
│   │   └── assessor.go          # Assessment mode: run matrix, collect metrics, generate report
│   ├── report/
│   │   └── markdown.go          # Comparison report generator (markdown table output)
│   └── tui/
│       ├── config.go            # Huh form: resolution, compression, subfolder prompts
│       ├── dashboard.go         # Bubbletea model: parallel progress bars
│       └── summary.go           # Lipgloss-styled results table
├── testdata/
│   └── samples/
│       └── LG-Daylight-4K-5s.mp4  # 5s 4K HEVC test clip
├── comparison_reports/          # Assessment output (gitignored)
├── Dockerfile                   # Multi-stage: Go build → Alpine + ffmpeg
├── docker-compose.yml           # Simple: single service, volume mount
├── justfile                     # build, test, run, docker-build, docker-run, assess
├── go.mod
├── go.sum
├── .gitignore
├── SPEC.md                      # Client brief
├── VC-Technical-Architecture.md # This document
├── VC-Processing-Pipeline.md    # ffmpeg integration details
└── VC-Phase0-Test-Plan.md       # Codec/CRF test matrix
```

---

## Docker Strategy

### Why Docker Is Required

Even though Go compiles to a static binary, the tool depends on ffmpeg/ffprobe at runtime. These have different builds, library versions, and codec support across Mac and Linux. Docker guarantees identical behaviour everywhere.

### Dockerfile

```dockerfile
# Stage 1: Build Go binary
FROM golang:1.24-alpine AS builder
WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -o /vc ./cmd/vc

# Stage 2: Runtime with ffmpeg
FROM alpine:3.21
RUN apk add --no-cache ffmpeg
COPY --from=builder /vc /usr/local/bin/vc
ENTRYPOINT ["vc"]
```

### Usage

```bash
# Build
docker build -t vc .

# Compress mode (interactive — TTY required for TUI)
docker run -it --rm -v /path/to/videos:/videos vc /videos

# Compress mode (non-interactive)
docker run -it --rm -v /path/to/videos:/videos vc compress /videos \
    --resolution 720p --compression high --recursive

# Assess mode — bind mount source videos + output dir
docker run -it --rm \
    -v /path/to/samples:/samples:ro \
    -v /path/to/reports:/reports \
    vc assess /samples --output /reports
```

**Bind mount strategy:**
- Source videos are mounted read-only (`:ro`) — the tool never modifies originals
- In compress mode, output goes next to source, so the mount must be read-write
- In assess mode, source can be read-only; a separate output mount receives the reports and encoded test files

### Docker Compose (Development)

```yaml
services:
  vc:
    build: .
    stdin_open: true
    tty: true
    volumes:
      - ${VIDEO_DIR:-./testdata}:/videos
      - ./comparison_reports:/reports
```

---

## Concurrency Model

### Goroutine Pool

The orchestrator runs N encoding jobs in parallel, where N defaults to `runtime.NumCPU() / 2` (ffmpeg itself is multi-threaded, so we avoid oversubscription).

```
                    ┌─────────────┐
                    │ Orchestrator│
                    └──────┬──────┘
                           │ fan-out
              ┌────────────┼────────────┐
              ▼            ▼            ▼
         ┌────────┐  ┌────────┐  ┌────────┐
         │ Worker │  │ Worker │  │ Worker │
         │ (Job1) │  │ (Job2) │  │ (Job3) │
         └───┬────┘  └───┬────┘  └───┬────┘
             │            │            │
             ▼            ▼            ▼
         ffmpeg        ffmpeg       ffmpeg
```

### Progress Flow (Elm Architecture)

```
ffmpeg stderr → parse progress → tea.Msg → Update() → View()
```

Each worker goroutine:
1. Spawns `ffmpeg` as a subprocess
2. Reads stderr line-by-line for `time=HH:MM:SS.ms` progress markers
3. Computes `progress = current_time / total_duration`
4. Sends a `ProgressMsg{JobID, Progress}` via the Bubbletea program channel
5. The TUI's `Update()` receives the message and updates the relevant progress bar
6. `View()` re-renders all progress bars

This is lock-free — no mutexes. The Bubbletea event loop serialises all state updates.

---

## CLI Interface

### Subcommands

```
vc compress /videos/           # Batch compress (default if no subcommand)
vc /videos/                    # Same — compress is default
vc assess /videos/samples/     # Run test matrix, generate comparison report
```

### Compress Mode — Interactive (Default)

When run with just a path, the TUI launches Huh forms:

```
$ vc /videos

┌─ Video Compressor ─────────────────────┐
│                                        │
│  Output Resolution:                    │
│  > 720p                                │
│    1080p                               │
│    4K                                  │
│                                        │
│  Compression Level:                    │
│  > High (smaller file, slower)         │
│    Low (larger file, faster)           │
│                                        │
│  Scan subfolders?                      │
│  > Yes                                 │
│    No                                  │
│                                        │
└────────────────────────────────────────┘
```

### Compress Mode — Flag Mode (Non-Interactive)

All prompts can be skipped with flags for scripting/CI:

```
vc /videos --resolution 720p --compression high --recursive
```

### Compress Flags

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--resolution` | `-r` | (prompt) | Target resolution: 720p, 1080p, 4k |
| `--compression` | `-c` | (prompt) | Compression level: high, low |
| `--recursive` | `-R` | (prompt) | Scan subfolders |
| `--workers` | `-w` | CPU/2 | Parallel encoding jobs |
| `--dry-run` | `-n` | false | Show what would be encoded, don't run |
| `--log-dir` | | stderr | Directory for log files |

### Assess Flags

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--output` | `-o` | `./comparison_reports/` | Report output directory |
| `--codecs` | | `h265,h264` | Codecs to test |
| `--crf-range` | | `23,26,28` (h265) / `20,23,25` (h264) | CRF values to test |
| `--presets` | | `slow,medium,fast` | Presets to test |
| `--resolutions` | | `source,1080p,720p` | Target resolutions |
| `--workers` | `-w` | 1 | Parallel jobs (default 1 for consistent timing) |

---

## Supported Input Formats

Accepted by extension (case-insensitive):

```
.mp4 .mkv .avi .mov .wmv .flv .webm .m4v .mpg .mpeg .3gp .ts
```

All output is `.mp4` (H.265/AAC or H.264/AAC depending on Phase 0 results).

---

## Error Handling

| Scenario | Behaviour |
|----------|-----------|
| ffmpeg not found | Fatal error with install instructions |
| Corrupt/unreadable video | Skip, log warning, continue batch |
| Output file exists | Skip (idempotent), log info |
| Disk full | Fail current job, clean up partial output, continue remaining |
| ffmpeg encode failure | Log error with ffmpeg stderr, continue batch |
| No videos found | Exit with message, code 0 |
| Ctrl+C | Graceful shutdown: kill ffmpeg subprocesses, clean up partial files |

---

## Testing Strategy

| Layer | Method | Dependencies |
|-------|--------|-------------|
| Domain (naming, profiles) | Unit tests | None |
| Ports | Interface compliance | None |
| ffmpeg adapter | Integration tests | ffmpeg binary + test fixtures |
| Orchestrator | Integration tests | ffmpeg binary + test fixtures |
| TUI | Manual testing | Terminal |
| Docker | `just docker-test` | Docker daemon |

Test fixtures: a small set of short (2-3 second) videos at various resolutions, codecs, and container formats. Committed to `testdata/`.

---

## Justfile

```just
default:
    @just --list

# Build the binary
build:
    go build -trimpath -o bin/vc ./cmd/vc

# Run tests
test:
    go test ./...

# Run compress mode with test data
run *ARGS:
    go run ./cmd/vc {{ARGS}}

# Run assessment on test samples
assess *ARGS:
    go run ./cmd/vc assess testdata/samples/ {{ARGS}}

# Build Docker image
docker-build:
    docker build -t vc .

# Run compress in Docker
docker-run *ARGS:
    docker run -it --rm -v $(pwd)/testdata:/videos vc /videos {{ARGS}}

# Run assessment in Docker
docker-assess *ARGS:
    docker run -it --rm -v $(pwd)/testdata/samples:/samples:ro -v $(pwd)/comparison_reports:/reports vc assess /samples --output /reports {{ARGS}}

# Lint
lint:
    golangci-lint run ./...

# Format
fmt:
    gofumpt -w .
```

---

## Open Decisions (Pending Phase 0 / Client Input)

| Decision | Options | Decided By |
|----------|---------|------------|
| Output codec | H.265 (libx265) vs H.264 (libx264) | Phase 0 size/quality comparison |
| CRF values | H.265: 23-28, H.264: 20-25 | Phase 0 results |
| Audio handling | Copy original vs re-encode AAC 128k | Phase 0 / client preference |
| Container format | MP4 (broad compat) vs MKV (flexible) | Client preference — likely MP4 |
| Preset mapping | high→slow/low→fast vs high→veryslow/low→medium | Phase 0 encode time analysis |
| Typical source formats | Phone H.264? Camera H.265? Screen recording? | Client — affects compression ratios |
| HDR handling | Tonemap to SDR or preserve HDR? | Client — test sample is HDR BT.2020 |

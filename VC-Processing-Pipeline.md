# VC Processing Pipeline

ffmpeg integration, encoding pipeline, and progress reporting architecture.

> **Status:** Draft — Codec/CRF values pending Phase 0 results.

---

## Pipeline Overview

```
Input Directory
      │
      ▼
┌─────────────┐     ┌──────────────┐     ┌──────────────┐     ┌─────────────┐
│  1. Scan    │────▶│  2. Probe    │────▶│  3. Plan     │────▶│  4. Encode  │
│  (discover) │     │  (metadata)  │     │  (schedule)  │     │  (ffmpeg)   │
└─────────────┘     └──────────────┘     └──────────────┘     └──────┬──────┘
                                                                      │
                                                                      ▼
                                                               ┌─────────────┐
                                                               │  5. Report  │
                                                               │  (summary)  │
                                                               └─────────────┘
```

### Stage 1: Scan (File Discovery)

**Port:** `Scanner`
**Adapter:** `fs.Scanner`

Walks the input directory (optionally recursive) collecting files with recognised video extensions.

```go
type Scanner interface {
    Scan(ctx context.Context, root string, recursive bool) ([]string, error)
}
```

Implementation uses `filepath.WalkDir` with an extension allowlist:

```go
var videoExtensions = map[string]bool{
    ".mp4": true, ".mkv": true, ".avi": true, ".mov": true,
    ".wmv": true, ".flv": true, ".webm": true, ".m4v": true,
    ".mpg": true, ".mpeg": true, ".3gp": true, ".ts": true,
}
```

Non-recursive mode uses `os.ReadDir` (single level) for efficiency.

Files are sorted by size descending — largest files start encoding first to minimise total wall-clock time when running parallel workers.

---

### Stage 2: Probe (Metadata Extraction)

**Port:** `Prober`
**Adapter:** `ffmpeg.Prober`

Extracts resolution, duration, codec, and file size from each discovered file using ffprobe.

```go
type Prober interface {
    Probe(ctx context.Context, path string) (domain.VideoMeta, error)
}
```

#### ffprobe Command

```bash
ffprobe -v quiet -print_format json -show_format -show_streams -select_streams v:0 INPUT
```

#### JSON Parsing

We extract from the first video stream (`streams[0]`):
- `width`, `height` → source resolution
- `codec_name` → source codec
- `duration` (from `format.duration`) → total duration for progress calculation

```go
type probeOutput struct {
    Streams []struct {
        Width     int    `json:"width"`
        Height    int    `json:"height"`
        CodecName string `json:"codec_name"`
    } `json:"streams"`
    Format struct {
        Duration string `json:"duration"`
        Size     string `json:"size"`
    } `json:"format"`
}
```

#### Probe Parallelism

Probing is I/O-bound and fast. All files are probed concurrently with a semaphore of 8 to avoid fd exhaustion.

---

### Stage 3: Plan (Job Scheduling)

**Pure domain logic — no external dependencies.**

For each probed file, the planner:

1. **Resolves effective resolution** — `min(source_height, target_height)`
2. **Builds encoding profile** — maps (resolution, compression_level) → ffmpeg parameters
3. **Computes output path** — applies naming convention
4. **Checks skip conditions** — output already exists, source matches target codec+resolution
5. **Creates Job** — ready for the encoding queue

#### Profile Matrix

These are the default profiles. CRF values are placeholders pending Phase 0.

| Resolution | Compression | Codec | CRF | Preset | Audio |
|-----------|-------------|-------|-----|--------|-------|
| 720p | High | libx265 | 26 | slow | aac 128k |
| 720p | Low | libx265 | 28 | fast | aac 128k |
| 1080p | High | libx265 | 24 | slow | aac 128k |
| 1080p | Low | libx265 | 26 | fast | aac 128k |
| 4K | High | libx265 | 22 | slow | aac 128k |
| 4K | Low | libx265 | 24 | fast | aac 128k |

> **Note:** If Phase 0 shows H.264 is significantly faster with acceptable size trade-off, we may offer it as an option or default for "Low" compression.

#### Scale Filter Logic

```go
func scaleFilter(sourceH, targetH int) string {
    if sourceH <= targetH {
        return "" // no scaling needed
    }
    // Scale to target height, preserve aspect ratio, ensure even dimensions
    return fmt.Sprintf("scale=-2:%d", targetH)
}
```

The `-2` ensures width is divisible by 2 (required by H.264/H.265 encoders).

---

### Stage 4: Encode (ffmpeg Execution)

**Port:** `Encoder`
**Adapter:** `ffmpeg.Encoder`

```go
type Encoder interface {
    Encode(ctx context.Context, job domain.Job, onProgress func(float64)) error
}
```

#### ffmpeg Command Construction

```bash
ffmpeg -y -i INPUT \
    -c:v libx265 -crf 24 -preset slow \
    -vf "scale=-2:720" \
    -c:a aac -b:a 128k \
    -movflags +faststart \
    -progress pipe:1 \
    OUTPUT.mp4
```

Key flags:
| Flag | Purpose |
|------|---------|
| `-y` | Overwrite output (we pre-check for existence, so this is safe) |
| `-c:v libx265` | Video codec |
| `-crf 24` | Constant Rate Factor — quality target |
| `-preset slow` | Encoding speed/compression trade-off |
| `-vf "scale=-2:720"` | Resolution scaling (omitted if no downscale needed) |
| `-c:a aac -b:a 128k` | Audio re-encode to AAC (or `-c:a copy` if source is already AAC) |
| `-movflags +faststart` | Move moov atom to start for streaming/progressive playback |
| `-progress pipe:1` | Machine-readable progress to stdout |

#### Audio Handling

```
Source audio is AAC → -c:a copy (no re-encode, preserves quality)
Source audio is other → -c:a aac -b:a 128k (re-encode)
```

This avoids unnecessary audio re-encoding when the source is already AAC.

---

### Progress Parsing

ffmpeg's `-progress pipe:1` outputs key-value pairs to stdout:

```
frame=150
fps=45.2
stream_0_0_q=28.0
bitrate=1250.5kbits/s
total_size=1048576
out_time_us=5000000
out_time_ms=5000000
out_time=00:00:05.000000
dup_frames=0
drop_frames=0
speed=1.5x
progress=continue
```

We parse `out_time_us` and divide by total duration (from probe) to get a 0.0–1.0 progress value.

```go
func parseProgress(line string, totalDuration time.Duration) (float64, bool) {
    if !strings.HasPrefix(line, "out_time_us=") {
        return 0, false
    }
    us, err := strconv.ParseInt(strings.TrimPrefix(line, "out_time_us="), 10, 64)
    if err != nil {
        return 0, false
    }
    current := time.Duration(us) * time.Microsecond
    return float64(current) / float64(totalDuration), true
}
```

The `progress=end` line signals completion.

#### Why `-progress pipe:1` Over Stderr Parsing

ffmpeg traditionally outputs progress to stderr in a human-readable format (`frame= 150 fps= 45`). The `-progress` flag provides structured key-value output that is far easier to parse reliably. We direct it to stdout (`pipe:1`) and read stderr separately for error messages.

---

### Stage 5: Report (Summary)

After all jobs complete, the reporter displays a styled summary table:

```
┌─────────────────────────────────────────────────────────────────┐
│                    Compression Complete                         │
├──────────────────┬──────────┬──────────┬─────────┬─────────────┤
│ File             │ Original │ Output   │ Saved   │ Status      │
├──────────────────┼──────────┼──────────┼─────────┼─────────────┤
│ holiday.mov      │ 1.2 GB   │ 245 MB   │ 79.6%   │ ✓ done      │
│ interview.mp4    │ 850 MB   │ 180 MB   │ 78.8%   │ ✓ done      │
│ clip.avi         │ 50 MB    │ —        │ —       │ ✗ failed    │
│ short.mp4        │ 12 MB    │ —        │ —       │ → skipped   │
├──────────────────┼──────────┼──────────┼─────────┼─────────────┤
│ Total            │ 2.1 GB   │ 425 MB   │ 79.5%   │ 2/4 encoded │
└──────────────────┴──────────┴──────────┴─────────┴─────────────┘
```

---

## Orchestration Detail

### Worker Pool Pattern

```go
func (o *Orchestrator) Run(ctx context.Context, jobs []domain.Job) []domain.Result {
    results := make([]domain.Result, len(jobs))
    sem := make(chan struct{}, o.workers)
    var wg sync.WaitGroup

    for i, job := range jobs {
        wg.Add(1)
        sem <- struct{}{} // acquire slot
        go func(idx int, j domain.Job) {
            defer wg.Done()
            defer func() { <-sem }() // release slot

            o.reporter.JobStarted(j)
            err := o.encoder.Encode(ctx, j, func(p float64) {
                o.reporter.JobProgress(j, p)
            })
            results[idx] = domain.Result{Job: j, Error: err}
            o.reporter.JobFinished(j, err)
        }(i, job)
    }

    wg.Wait()
    return results
}
```

### Bubbletea Integration

The TUI model acts as the reporter. The orchestrator runs in a goroutine launched via `tea.Cmd`. Progress updates flow as Bubbletea messages:

```go
type JobStartedMsg   struct{ JobID int }
type JobProgressMsg  struct{ JobID int; Progress float64 }
type JobFinishedMsg  struct{ JobID int; Result domain.Result }
type AllDoneMsg      struct{ Results []domain.Result }
```

The `Update()` method handles these messages to update the dashboard. The `View()` method renders N progress bars (one per active job) plus a summary of completed/failed jobs.

---

## Graceful Shutdown

On `SIGINT`/`SIGTERM` or Bubbletea's `tea.KeyCtrlC`:

1. Cancel the context passed to the orchestrator
2. Each worker checks `ctx.Done()` and kills its ffmpeg subprocess (`cmd.Process.Kill()`)
3. Workers clean up partial output files (incomplete `.mp4` files)
4. Orchestrator collects results (some done, some cancelled)
5. Summary displays what completed and what was cancelled

---

## File I/O Safety

| Concern | Mitigation |
|---------|-----------|
| Filenames with spaces | All paths passed to ffmpeg are properly quoted/escaped via `exec.Command` (no shell expansion) |
| Unicode filenames | Go's `os` and `filepath` handle UTF-8 natively |
| Long paths | Not an issue on Linux/Mac (PATH_MAX=4096) |
| Partial writes | ffmpeg writes to the final path; if killed, we delete the partial file |
| Symlinks | `filepath.WalkDir` follows symlinks by default — acceptable for this use case |
| Permissions | If output dir is read-only, ffmpeg fails and we report the error per-job |

---

## Configuration Precedence

```
1. CLI flags          (highest priority)
2. Interactive prompts (if flags not provided)
3. Defaults           (workers=CPU/2, etc.)
```

There is no config file. The tool is intentionally simple — run it, answer 3 questions, done.

---

## Dependency Graph

```
cmd/vc/main.go
  ├── internal/tui          (Bubbletea models)
  │     ├── internal/domain  (types)
  │     └── internal/app     (orchestrator)
  │           ├── internal/port     (interfaces)
  │           ├── internal/adapter/ffmpeg
  │           └── internal/adapter/fs
  └── internal/domain        (types — no dependencies)
```

The domain package has zero imports outside stdlib. Ports depend only on domain. Adapters implement ports and may import external packages (os/exec for ffmpeg). The TUI depends on app and domain but never on adapters directly — it receives them through the orchestrator.

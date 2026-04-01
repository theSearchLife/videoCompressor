# Video Compressor

A Docker-first video compression tool.

## Current Status

The repository currently supports:

- Docker-only execution
- recursive scanning of mixed-content directories
- `.mp4` compression outputs
- `.tmp` output paths plus atomic rename on success
- internal assessment runs through `vc assess`

### Project Roadmap

| Phase | What | Status |
|-------|------|--------|
| **Requirements** | Confirm source-video and output expectations | **Active now** |
| **Phase 0 Assessment** | `vc assess` produces comparison outputs and report files | **Ready for review** |
| **Final Compression Workflow** | Lock profiles and complete the full production UX | Next |

## What Works Today

### Phase 0 assessment
- `vc assess /samples --output /reports`
- Generates encoded comparison files in a timestamped folder
- Generates `report.md` and `results.csv` in that same folder
- Lets us compare size, speed, and optional VMAF before locking settings

### Basic compression command
- `vc compress /videos --resolution 720p --compression high`
- Docker-only run path works; no local Go install is required

### Current guarantees
- Never modifies or deletes original files
- Won't upscale: if a video is already 720p and you pick 1080p, it stays at 720p
- Scans subfolders automatically
- Deletes stale `*.tmp` files during scan
- Writes to `<final>.tmp` during encode and renames to final `.mp4` on success
- Adds a suffix: `holiday.mp4` → `holiday_720p.mp4`
- Scans common video formats such as `.mp4`, `.mov`, `.mkv`, `.avi`, `.webm`, and `.ts`

### Not promised yet
- Interactive prompts
- Final polished client UX for the full batch-compression phase
- Final codec/profile choices before client sample review

## Docker-Only Quick Start

You only need [Docker](https://docs.docker.com/get-docker/). No local Go install is needed.

### 1. Build the image (first time only)

```bash
docker build -t vc .
```

### 2. Put sample videos in the input folder

Use `testdata/samples/` for local review. If you need a free sample clip, see [testdata/samples/README.md](testdata/samples/README.md).

### 3. Check the input folder without encoding

```bash
docker run --rm -v $(pwd)/testdata/samples:/videos vc compress /videos \
    --resolution 720p --compression high --dry-run
```

This prints which files would be processed and where outputs would go.

### 4. Run the assessment flow

```bash
docker run --rm \
    -v $(pwd)/testdata/samples:/samples:ro \
    -v $(pwd)/comparison_reports:/reports \
    vc assess /samples --output /reports
```

Assessment outputs appear here:
- Input files: `testdata/samples/`
- Encoded comparison files: `comparison_reports/<timestamp>/encoded/`
- Summary report: `comparison_reports/<timestamp>/report.md`
- CSV data: `comparison_reports/<timestamp>/results.csv`

### 5. Optional direct compression

```bash
docker run --rm -v $(pwd)/testdata/samples:/videos vc compress /videos \
    --resolution 720p --compression high
```

### Windows Docker examples

```powershell
docker run --rm -v "C:\Videos:/videos" vc compress /videos --resolution 720p --compression high --dry-run
```

```powershell
docker run --rm -v "C:\Videos:/samples:ro" -v "C:\Reports:/reports" vc assess /samples --output /reports
```

Compression output appears next to the original file:
- `clip.mp4` → `clip_720p.mp4`

### Using `just` (optional shortcut)

If you have [`just`](https://github.com/casey/just) installed:

```bash
just build              # Build Docker image
just run --resolution 720p --compression high   # Compress testdata/samples/
just assess             # Run quality comparison matrix
```

`just` also runs through Docker, so it still does not require a local Go install.

## Client Review Flow

1. Answer the questions in [VC-Client-Questions.md](VC-Client-Questions.md)
2. Share 2-3 real sample videos if possible
3. Run or review `vc assess`
4. Open the generated comparison folder
5. Confirm which outputs are good enough
6. Lock the final compression profile for the next phase

## Project Files

| File | What's in it |
|------|-------------|
| [VC-Client-Questions.md](VC-Client-Questions.md) | **Start here**: plain-language questions for the client |
| [SPEC.md](SPEC.md) | Original job brief and agreed approach |
| [VC-Phase0-Test-Plan.md](VC-Phase0-Test-Plan.md) | Assessment plan and output structure |
| [VC-Technical-Architecture.md](VC-Technical-Architecture.md) | Internal architecture (for developer reference) |
| [VC-Processing-Pipeline.md](VC-Processing-Pipeline.md) | How the encoding pipeline works (for developer reference) |

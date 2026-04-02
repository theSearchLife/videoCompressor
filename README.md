# Video Compressor

A Docker-first CLI tool for batch video compression with H.265/HEVC.

## Quick Start

You only need [Docker](https://docs.docker.com/get-docker/). No local Go or ffmpeg install required.

### 1. Build the image (first time only)

The `vc` script builds the image automatically on first run. Or build it manually:

```bash
docker build -t vc .
```

### 2. Compress videos

Point `vc` at a folder. It scans all subfolders, picks out video files, and skips everything else.

**Linux / macOS:**
```bash
./vc /mnt/videos
```

**Windows (PowerShell):**
```powershell
.\vc.ps1 C:\Videos
```

The tool prompts for six settings interactively:

```
Compression strategy:
  1. Quality priority
  2. Keep quality and reduce size (default)
  3. Size priority

Resolution:
  1. Keep original (default)
  2. 720p
  3. 1080p
  4. 4k

Frame rate:
  1. Keep original (default)
  2. 24 fps
  3. 30 fps
  4. 60 fps

Audio quality:
  1. Keep original (default)
  2. Low (96 kbps)
  3. Medium (128 kbps)
  4. High (192 kbps)

Output suffix (default: _compressed):
> _compressed

Skip already converted?
  1. Yes (skip files that already have a converted output) (default)
  2. No (re-encode everything)
```

Or pass flags to skip the prompts:

```bash
./vc /mnt/videos --strategy balanced --resolution 1080p --suffix _compressed
```

Output appears next to the original file:
- `holiday.mov` → `holiday_compressed.mp4`

### 3. Compare different settings

Use different suffixes to try different compression settings on the same files. Each suffix produces a separate output, so results coexist for side-by-side comparison.

```bash
# Try quality priority
./vc /mnt/camera/shoot1 --strategy quality --suffix _v1

# Try size priority
./vc /mnt/camera/shoot1 --strategy size --suffix _v2

# Compare clip_v1.mp4 vs clip_v2.mp4, then clean up the one you prefer
./vc cleanup /mnt/camera/shoot1 --suffix _v2
```

### 4. Preview without encoding

```bash
./vc /mnt/videos --dry-run
```

### 5. Resume after interruption

If a run is interrupted, just run the same command again. It cleans up incomplete `.tmp` files, skips files that already have a converted output, and picks up where it left off.

To force re-encoding of everything (e.g. after changing your mind on settings), use `--skip-converted no`.

### 6. Clean up originals after review

Once you have reviewed the compressed outputs and are happy with the quality:

```bash
./vc cleanup /mnt/videos
```

The tool prompts for the suffix used during compression and asks for confirmation:

```
Output suffix (default: _compressed):
> _compressed

Found 3 originals with matching converted outputs:
  holiday.mov -> holiday.mp4
  clip.avi -> clip.mp4
  recording.mkv -> recording.mp4

Delete originals and rename converted files?
  1. Yes
  2. No (default)
>
```

This deletes the original and renames the compressed output:
- Deletes `holiday.mov`
- Renames `holiday_compressed.mp4` → `holiday.mp4`

## Behaviour

- Never modifies or deletes original files during compression
- Never upscales: if a video is already 720p and you pick 1080p, it stays at 720p
- Scans subdirectories automatically, skipping non-video files
- Writes to `.tmp` during encode, atomically renames to `.mp4` on success
- Skips completed conversions on rerun — safe to interrupt and resume
- Accepts `.mp4`, `.mov`, `.mkv`, `.avi`, `.wmv`, `.webm`, `.mpg`, `.mpeg`, `.m4v`, `.flv`, `.3gp`, `.ts`
- Handles filenames with spaces, special characters, and unicode
- All output uses `.mp4` container format

## CLI Reference

```
./vc <input-dir> [flags]                 Compress videos
./vc cleanup <input-dir> [flags]         Delete originals and rename converted outputs
./vc assess <input-dir> [flags]          Run codec/CRF test matrix

Compress flags:
  --strategy        quality|balanced|size   Compression strategy (default: balanced)
  --resolution      original|720p|1080p|4k  Target resolution (default: original)
  --fps             0|24|30|60              Frame rate, 0=keep original (default: 0)
  --audio           keep|low|medium|high    Audio quality (default: keep)
  --suffix          STRING                  Output file suffix (default: _compressed)
  --skip-converted  yes|no                  Skip already converted files (default: yes)
  --workers N                               Parallel jobs (default: CPU/2)
  --dry-run                                 Show what would be encoded

Cleanup flags:
  --suffix          STRING                  Suffix used during compression (default: _compressed)
```

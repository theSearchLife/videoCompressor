# Video Compressor

A Docker-first batch video compression tool for H.265/HEVC.

## Quick Start

You only need [Docker](https://docs.docker.com/get-docker/). No local Go or ffmpeg install required.

```bash
docker run --rm -it -v /path/to/videos:/videos ghcr.io/thesearchlife/videocompressor:main /videos
```

**Windows (PowerShell):**
```powershell
docker run --rm -it -v C:\Videos:/videos ghcr.io/thesearchlife/videocompressor:main /videos
```

To see all available options:

```bash
docker run --rm ghcr.io/thesearchlife/videocompressor:main -h
```

### Compress videos

Point the container at a folder. It scans all subfolders, picks out video files, and skips everything else.

The tool prompts for six settings interactively:

```
Compression strategy:
  1. Quality (slow)
  2. Balanced (default)
  3. Size (fast)

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

Output appears next to the original file:
- `holiday.mov` → `holiday_compressed.mp4`

### Resume after interruption

If a run is interrupted, just run the same command again. It cleans up incomplete `.tmp` files, skips files that already have a converted output, and picks up where it left off.

### Clean up originals after review

Once you have reviewed the compressed outputs and are happy with the quality:

```bash
docker run --rm -it -v /path/to/videos:/videos ghcr.io/thesearchlife/videocompressor:main cleanup /videos
```

The tool prompts for the suffix used during compression and asks for confirmation before deleting originals and renaming compressed outputs.

## Development

For end users, Docker is the only prerequisite. Do not install Homebrew, Go,
ffmpeg, ffprobe, or `just` to run the compressor.

Maintainers can optionally use `just` as a shortcut for Docker-backed
development tasks:

```bash
just --list
```

## Behaviour

- Never modifies or deletes original files during compression
- Never upscales: if a video is already 720p and you pick 1080p, it stays at 720p
- Scans subdirectories automatically, skipping non-video files
- Writes to `.tmp` during encode, atomically renames to `.mp4` on success
- Skips completed conversions on rerun — safe to interrupt and resume
- Accepts `.mp4`, `.mov`, `.mkv`, `.avi`, `.wmv`, `.webm`, `.mpg`, `.mpeg`, `.m4v`, `.flv`, `.3gp`, `.ts`
- Handles filenames with spaces, special characters, and unicode
- All output uses `.mp4` container format

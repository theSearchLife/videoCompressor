# Job Specification

Source: Upwork contract offer from Krasi Georgiev (2026-03-30)
Budget: $100 fixed price
Repo: https://github.com/theSearchLife/videoCompressor

## Brief

Golang-based CLI tool OR Bash script for Linux and a Docker-based way to run it on Mac and Linux for batch video compression.

The goal is to create a tool that reduces video file sizes as much as possible while keeping good quality.

## Requirements

### If implemented using Golang (preferred):
- A Golang-based CLI tool
- Cross-platform support (Linux, Mac, Windows if possible)

### If implemented using Bash:
1. Bash script for Linux — processes video files with interactive prompts
2. Docker support — must be runnable via `docker run` on both Mac and Linux

### Docker usage style:
```
docker run -it -v /home/videos:/videos containerName
```

The developer should also provide clear command instructions for running it.

## Script Flow / Prompts

The tool should ask the user for:

**Output resolution:**
- 720p
- 1080p
- 4K

**Compression level:**
- High (slow)
- Low (fast)

**Scan subfolders?**
- Yes / No

## Functional Requirements

- Aim for minimum possible file size
- Add a suffix to each converted file based on the chosen resolution (e.g. `vid1.mp4` -> `vid1_720p.mp4`)
- Do not upscale — if a video is already 720p and output is set to 1080p, it must remain 720p
- Must support batch conversion
- Should work reliably with common video formats

## Deliverables

- Golang-based CLI tool or Bash script with Docker support for batch video compression
- Cross-platform support (Linux, Mac, Windows if possible)
- Simple usage instructions
- Example commands
- Clear explanation of any dependencies or assumptions
- Dockerfile / Docker image setup (optional, if used)

## Nice to Have

- Clean logging/output
- Sensible defaults
- Safe handling of filenames with spaces
- Option to preserve original files while saving converted versions separately

## Agreed Approach

1. **Requirements confirmation** — container format (mp4/mkv), target codec, resolution options, how input arrives (batch folder vs individual files)
2. **Phase zero** — run sample videos through a matrix of codec/CRF/resolution settings, deliver outputs with a comparison table for client sign-off
3. **Implementation** — once settings are locked, build the Go binary with directory walker, goroutine-based parallel encoding, progress output, Docker packaging

## Client Notes

- Docker is required even with Go — backend tools (ffmpeg etc.) differ between Mac and Linux; container should bundle all dependencies rather than relying on the host
- Kaloyan Georgiev handled initial candidate selection; Krasi Georgiev is the primary technical contact and contract owner

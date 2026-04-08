# Job Specification

Budget: $100 fixed price
Repo: https://github.com/theSearchLife/videoCompressor

## Brief

Build a Go CLI tool for batch video compression. Deliver the runtime through Docker for Linux, macOS, and Windows users.

The tool compresses videos with H.265/HEVC by default, reduces file size while preserving acceptable quality, and operates on a mounted host directory.

## Input Profile

- Source footage is a mix of phone videos and Sony ZV-1E S-Log3 videos.
- Typical file size is 50 MB to 10 GB.
- Rare file size is up to 30 GB.
- Typical batch size is 5 to 10 videos.
- Large batch size is up to 100 videos.
- Rare batch size is up to 300 videos.
- Files are spread across nested subfolders.
- Source directories contain mixed content such as images and documents.
- Audio is a single stereo track.
- Subtitles are not part of scope.
- Playback targets are VLC on a laptop and Adobe Premiere Pro for editing.

## Runtime Model

- Docker is the execution path on Linux, macOS, and Windows.
- The container bundles the Go binary, `ffmpeg`, and `ffprobe`.
- The tool reads from a bind-mounted host directory.
- The tool writes outputs back into the mounted host directory.
- Windows support means Docker Desktop plus Windows bind mounts. It does not mean a native Windows install flow.

## User Flow

The tool scans the mounted directory recursively and processes only recognised video files.

The compression flow prompts for:

1. Compression strategy
   `Quality (slow)`, `Balanced`, `Size (fast)`
2. Resolution
   `Keep original` or reduce to a specified resolution
3. Frame rate
   `Keep original` or reduce to a specified rate
4. Audio quality
   `Keep original`, `Low`, `Medium`, `High`
5. File suffix
   Example: `clip.mov` -> `clip_compressed.mp4`
6. Delete originals
   Second-pass cleanup after review

## Output Policy

- Final outputs are written next to the source files.
- Final outputs always use the `.mp4` container.
- Compression outputs use the chosen suffix before `.mp4`.
- Temporary outputs use the final output path plus `.tmp`.
  Example: `clip_1080p.mp4.tmp`
- On successful encode, the tool atomically renames the temporary file to the final `.mp4` path.
- On each scan, the tool deletes stale `*.tmp` files before planning work.

## Compression Rules

- Default video codec is H.265/HEVC.
- Resolution never upscales.
- If the source is below the requested target resolution, the source resolution is preserved.
- Common video formats are accepted as inputs, including `mp4`, `mkv`, `avi`, `mov`, `wmv`, and `webm`.
- Filenames with spaces and special characters are supported.
- Progress is logged per file.

## Skip And Resume Rules

- Completed conversions are skipped on rerun.
- Skip detection uses the configured suffix and the expected final `.mp4` output path.
- Interrupted or failed runs leave incomplete work in `.tmp` paths.
- The next scan removes stale `.tmp` files and resumes remaining work.

## Cleanup Rules

- Cleanup is a second-pass flow after review.
- If both the original file and the converted suffixed `.mp4` file exist, cleanup deletes the original file.
- Cleanup then atomically renames the suffixed `.mp4` file to the unsuffixed `.mp4` filename.
- Example:
  `clip.mov` + `clip_compressed.mp4` -> delete `clip.mov`, rename `clip_compressed.mp4` to `clip.mp4`

## Deliverables

- Go CLI tool for batch video compression
- Dockerfile for the runtime image
- Single entry-point script that builds the image if needed and launches the container
- README with Linux, macOS, and Windows Docker usage examples

## Internal Validation

- Phase 0 uses `vc assess` as internal tooling to compare codec, CRF, preset, and resolution combinations against client sample videos.
- Phase 0 produces encoded comparison files plus a report for sign-off.
- `vc assess` is internal validation tooling, not a client deliverable requirement.

## Out Of Scope

- Scheduled or automated execution
- Multiple audio tracks
- Subtitle handling
- Native Windows installation outside Docker

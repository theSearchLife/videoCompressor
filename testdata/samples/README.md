# Test Samples

Put sample input videos in this folder when testing locally.

## Quick Start

1. Put one or more video files in `testdata/samples/`
2. Build the Docker image
3. Run either a dry-run check or the assessment command

```bash
just build                    # Build the Docker image (first time only)
docker run --rm -v $(pwd)/testdata/samples:/videos vc compress /videos --resolution 720p --compression high --dry-run
```

## Demo Sample (Optional)

If you don't have a video handy, you can download the LG 4K demo clip we used for testing:

1. Go to https://www.demolandia.net/downloads.html?id=556436627
2. Download the MP4 version
3. Save it to this directory as `LG-Daylight-4K.mp4`

Or use any video file you have — the tool accepts `.mp4`, `.mov`, `.mkv`, `.avi`, `.webm`, and other common formats.

## Input And Output

- Input folder for local testing:
  `testdata/samples/`

- Dry-run command:
  Reads the input folder and shows what would be created without encoding anything.

- Compress mode output:
  Output appears next to the original source file in this same directory.
  `clip.mp4` → `clip_720p.mp4`

- Assess mode output:
  Output goes to `comparison_reports/<timestamp>/` in the project root.
  That folder contains:
  `encoded/`
  `report.md`
  `results.csv`

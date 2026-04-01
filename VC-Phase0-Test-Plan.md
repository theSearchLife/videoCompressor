# VC Phase 0 Test Plan

## Purpose

Phase 0 locks the compression profile against real client samples before the full client-facing workflow is finalised.

Phase 0 is internal validation. It uses `vc assess` to generate comparison outputs and a report for review.

## Inputs

- Client sample videos live in a local untracked directory.
- Samples must include:
  phone footage,
  Sony ZV-1E S-Log3 footage,
  and a lower-resolution clip to confirm no-upscale behaviour.

## Decisions Locked By Phase 0

- Codec choice
- CRF range
- Preset mapping
- Resolution mapping
- Audio handling defaults

## Test Matrix

### Codec

- `libx265`
- `libx264`

### CRF

- H.265: `23`, `26`, `28`
- H.264: `20`, `23`, `25`

### Preset

- `slow`
- `medium`
- `fast`

### Resolution Set

- source resolution
- `1080p`
- `720p`

### Audio Set

- copy source audio
- AAC `128k`
- AAC `96k`

## Execution

### Local

```bash
vc assess /path/to/client-samples --output /path/to/reports
```

### Docker

```bash
docker run --rm \
    -v /path/to/client-samples:/samples:ro \
    -v /path/to/reports:/reports \
    vc assess /samples --output /reports
```

## Output Structure

```text
comparison_reports/<timestamp>/
├── encoded/
├── report.md
└── results.csv
```

- `encoded/` contains temporary comparison artefacts for review.
- `report.md` contains the comparison summary and recommendation.
- `results.csv` contains machine-readable results.

## Metrics

- input size
- output size
- reduction percentage
- encode time
- encode speed
- output bitrate
- VMAF score when available

VMAF is advisory only. It is not a release gate for HDR, log, or downscaled content.

## Sign-Off Criteria

- visual quality is acceptable on the client’s real footage
- file size reduction is acceptable
- encode speed is acceptable
- no-upscale behaviour is correct
- output container and playback behaviour are acceptable for VLC and Premiere Pro

# VC Phase 0 — Test Plan

Codec/CRF/resolution test matrix for client sign-off before implementation begins.

> **Status:** Ready to execute — test sample available (`testdata/samples/LG-Daylight-4K-5s.mp4`).

---

## Objective

Run sample videos through a matrix of encoding settings. Deliver encoded variants with structured filenames and a comparison report so the client can evaluate quality vs size trade-offs and sign off on the target profile.

The assessment system (`vc assess`) is a first-class feature — the client can run it themselves on their own hardware and samples.

### What We're Deciding

| Decision | Options Under Test |
|----------|--------------------|
| **Codec** | H.265 (libx265) vs H.264 (libx264) |
| **CRF range** | H.265: 23, 26, 28 / H.264: 20, 23, 25 |
| **Preset** | slow, medium, fast |
| **Audio** | Copy vs re-encode AAC 128k |
| **Container** | MP4 (assumed — MKV only if client requests) |
| **Source format handling** | How well does the pipeline handle various input codecs/containers? |

---

## Test Samples

### Available Now

| Sample | Properties | Notes |
|--------|-----------|-------|
| `LG-Daylight-4K-5s.mp4` | 3840x2160, HEVC Main 10, HDR (BT.2020/PQ), 60fps, 5s, 38 MB | Trimmed from [demolandia.net LG 4K demo](https://www.demolandia.net/downloads.html?id=556436627). Client can download the full 48s source to test. |

### Desired (Client to Provide)

We need to know the client's typical source formats. The test sample is 4K HEVC 10-bit HDR — a demanding edge case. Additional samples would strengthen the assessment:

| Sample | Description | Why |
|--------|-------------|-----|
| Phone recording (H.264) | 1080p, 8-bit SDR | Most common consumer source |
| Screen recording | Variable resolution, potentially low-motion | Very different compression profile |
| 720p source | Any codec | Validates no-upscale logic |

**Source type is a key client question** — their typical input codecs/containers will significantly affect compression ratios and encoding speed.

---

## Test Matrix

### Codec × CRF × Preset

Each sample is encoded through every combination:

| # | Codec | CRF | Preset | Target Res | Expected Trade-off |
|---|-------|-----|--------|------------|-------------------|
| 1 | libx265 | 23 | slow | 1080p | Best quality, smallest size, slowest |
| 2 | libx265 | 23 | medium | 1080p | Good quality, medium speed |
| 3 | libx265 | 23 | fast | 1080p | Good quality, fastest H.265 |
| 4 | libx265 | 26 | slow | 1080p | Balanced quality/size |
| 5 | libx265 | 26 | medium | 1080p | **Expected sweet spot** |
| 6 | libx265 | 26 | fast | 1080p | Balanced, faster |
| 7 | libx265 | 28 | slow | 1080p | Aggressive compression |
| 8 | libx265 | 28 | medium | 1080p | Aggressive, medium speed |
| 9 | libx265 | 28 | fast | 1080p | Most aggressive H.265 |
| 10 | libx264 | 20 | slow | 1080p | Best H.264 quality |
| 11 | libx264 | 20 | medium | 1080p | Good H.264 quality |
| 12 | libx264 | 23 | slow | 1080p | Balanced H.264 |
| 13 | libx264 | 23 | medium | 1080p | **H.264 sweet spot candidate** |
| 14 | libx264 | 23 | fast | 1080p | Balanced H.264, faster |
| 15 | libx264 | 25 | slow | 1080p | Aggressive H.264 |
| 16 | libx264 | 25 | medium | 1080p | Aggressive H.264, medium |
| 17 | libx264 | 25 | fast | 1080p | Most aggressive H.264 |

Additionally for the 4K and 720p samples, a subset of the best-performing settings applied at their native + downscaled resolutions.

### Resolution Tests (Subset)

Run with the best CRF/preset from each codec (determined from the 1080p matrix):

| # | Source | Target | Codec | CRF | Preset |
|---|--------|--------|-------|-----|--------|
| R1 | 4K | 4K | best_h265 | best | best |
| R2 | 4K | 1080p | best_h265 | best | best |
| R3 | 4K | 720p | best_h265 | best | best |
| R4 | 720p | 1080p | best_h265 | best | best |
| R5 | 720p | 720p | best_h265 | best | best |

R4 validates the no-upscale rule (output should remain 720p).

### Audio Test

| # | Audio Treatment | Command Diff |
|---|----------------|-------------|
| A1 | Copy original | `-c:a copy` |
| A2 | Re-encode AAC 128k | `-c:a aac -b:a 128k` |
| A3 | Re-encode AAC 96k | `-c:a aac -b:a 96k` |

Run on a single sample with the winning video settings. Compare file sizes — audio is typically <10% of total size so re-encoding rarely matters much, but if the source is already AAC, copy is free quality.

---

## Execution

### Assessment Command

The assessment is a built-in subcommand, not a separate script:

```bash
# Run locally
vc assess testdata/samples/

# Run via Docker (client can do this)
docker run -it --rm \
    -v /path/to/samples:/samples:ro \
    -v /path/to/reports:/reports \
    vc assess /samples --output /reports

# Via justfile
just assess
```

### Output Structure

```
comparison_reports/
└── 2026-03-31T14-30-00/              # Timestamped run
    ├── encoded/                        # All encoded variants (temp — delete after review)
    │   ├── LG-Daylight-4K-5s_h265_crf23_slow_2160p.mp4
    │   ├── LG-Daylight-4K-5s_h265_crf23_slow_1080p.mp4
    │   ├── LG-Daylight-4K-5s_h265_crf26_medium_1080p.mp4
    │   ├── LG-Daylight-4K-5s_h264_crf23_medium_1080p.mp4
    │   └── ...
    ├── report.md                       # Comparison tables + recommendation
    └── results.csv                     # Machine-readable metrics
```

The structured filename format `{source}_{codec}_crf{N}_{preset}_{resolution}.mp4` lets the client identify each variant at a glance in a file browser, sorted naturally by codec then CRF.

### ffmpeg Command Template

```bash
ffmpeg -y -i INPUT \
    -c:v ${CODEC} -crf ${CRF} -preset ${PRESET} \
    -vf "scale=-2:${HEIGHT}" \
    -c:a copy \
    -movflags +faststart \
    -progress pipe:1 \
    output/${SAMPLE}_${CODEC}_crf${CRF}_${PRESET}_${RES}.mp4
```

### Metrics Collected Per Encode

| Metric | Source | Purpose |
|--------|--------|---------|
| Input file size | `os.Stat` | Baseline |
| Output file size | `os.Stat` | Primary metric — smaller is better |
| Compression ratio | `output / input` | Normalised comparison |
| Size reduction % | `1 - (output / input)` | Client-friendly metric |
| Encode time | Wall clock | Speed trade-off |
| Encode speed | `duration / encode_time` | e.g. "2.5x realtime" |
| Output bitrate | ffprobe | Sanity check |
| VMAF score (optional) | `ffmpeg -lavfi libvmaf` | Perceptual quality metric (0-100) |

### VMAF (Optional but Recommended)

VMAF is Netflix's perceptual video quality metric. It compares the encoded output against the source frame-by-frame and produces a score from 0-100, where:
- **93+** = visually lossless (indistinguishable from source)
- **85-93** = good quality (minor differences visible on close inspection)
- **75-85** = acceptable quality (noticeable on large screens)
- **<75** = quality loss visible to casual viewers

```bash
ffmpeg -i OUTPUT -i INPUT \
    -lavfi libvmaf="model=version=vmaf_v0.6.1" \
    -f null -
```

> **Note:** VMAF requires the `libvmaf` library. It's available in the Docker image (Alpine's ffmpeg package includes it). If not available on the host, we skip it and rely on visual inspection + file size comparison.

---

## Comparison Table Template

This is what we deliver to the client for sign-off:

### Sample: `sample_1080p_high.mp4` (Original: 150 MB, 1080p, 60s)

| # | Codec | CRF | Preset | Output Size | Reduction | Encode Time | Speed | VMAF |
|---|-------|-----|--------|-------------|-----------|-------------|-------|------|
| 1 | H.265 | 23 | slow | ? MB | ?% | ?s | ?x | ? |
| 2 | H.265 | 23 | medium | ? MB | ?% | ?s | ?x | ? |
| 3 | H.265 | 23 | fast | ? MB | ?% | ?s | ?x | ? |
| 4 | H.265 | 26 | slow | ? MB | ?% | ?s | ?x | ? |
| 5 | H.265 | 26 | medium | ? MB | ?% | ?s | ?x | ? |
| ... | ... | ... | ... | ... | ... | ... | ... | ... |

### Recommendation Row

After filling in the table, we highlight:

```
RECOMMENDED PROFILES:
  High compression: H.265, CRF ?, preset slow  — ?% reduction, VMAF ?, ?x speed
  Low compression:  H.265, CRF ?, preset fast   — ?% reduction, VMAF ?, ?x speed
```

---

## Client Sign-Off Criteria

The client needs to approve:

1. **Quality gate** — Which VMAF score / visual quality is the minimum acceptable?
   - Suggestion: VMAF ≥ 85 for "High" compression, VMAF ≥ 90 for "Low" compression
   - Or: "I've watched the outputs and X looks good to me"

2. **Codec choice** — H.265 vs H.264
   - H.265: ~30-40% smaller files, slower encoding, less universal playback
   - H.264: Larger files, faster encoding, plays everywhere
   - Recommendation: H.265 (the brief says "reduce file sizes as much as possible")

3. **Compression level mapping** — What "High" and "Low" mean in practice
   - High = slow preset + lower CRF (best compression, 0.5-1x realtime)
   - Low = fast preset + higher CRF (quick encode, 3-5x realtime)

4. **Audio handling** — Copy vs re-encode
   - Recommendation: Copy if source is AAC, re-encode otherwise

5. **Container format** — MP4 confirmed (or MKV if client has a reason)

---

## Deliverable

A timestamped directory under `comparison_reports/`:

```
comparison_reports/2026-03-31T14-30-00/
├── encoded/                                                # Temp — delete after review
│   ├── LG-Daylight-4K-5s_h265_crf23_slow_2160p.mp4
│   ├── LG-Daylight-4K-5s_h265_crf23_slow_1080p.mp4
│   ├── LG-Daylight-4K-5s_h265_crf26_medium_1080p.mp4
│   ├── ...
│   └── LG-Daylight-4K-5s_h264_crf25_fast_720p.mp4
├── report.md                                               # Comparison tables + recommendation
└── results.csv                                             # Machine-readable metrics
```

**Client workflow:**
1. Run `vc assess /their/samples/` (locally or via Docker)
2. Browse `encoded/` folder — filenames describe each variant
3. Watch files to judge visual quality
4. Read `report.md` for size/speed/VMAF comparison tables
5. Confirm or adjust recommended profiles
6. We lock in profiles and proceed to full implementation
7. Delete `encoded/` to reclaim disk space

---

## Client Can Self-Serve

The assessment is a built-in `vc assess` command. The client can:
- Run it on their own hardware with their own source videos
- Customise the matrix via flags (`--codecs`, `--crf-range`, `--presets`, `--resolutions`)
- Compare results across different machines (encode speed will differ, quality metrics won't)

This means Phase 0 is not a one-shot deliverable — it's a reusable tool.

---

## Timeline

| Step | Effort |
|------|--------|
| Build assessment prototype | Implementation work — the first deliverable |
| Run on test sample | Automated — `just assess` |
| Create GitHub issue with client questions | After first results are in |
| Client runs assessment on their samples | Blocked on client providing source files |
| Lock profiles, build compress mode | After client sign-off |

Phase 0 execution is fully automated. The main blocking dependency is knowing the client's typical source formats.

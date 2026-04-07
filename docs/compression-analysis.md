# Compression Analysis

Analysis of client-provided source files and compression results.
Conducted 2026-04-07.

## Source Files

### 20210926_100615_1(short).mp4

This is the file the client tested as "ORIGINAL.mp4" in their test document.

| Property       | Value                          |
|----------------|--------------------------------|
| Size           | 25.9 MiB                      |
| Duration       | 15.1 seconds                   |
| Codec          | H.264 High (avc1)             |
| Resolution     | 1920x1080                      |
| Frame rate     | ~30 fps                        |
| Video bitrate  | ~14.1 Mbps                     |
| Audio          | AAC ~252 kbps                  |
| Source device  | Phone (2021, portrait rotation)|

### 20190903_140907.mp4

| Property       | Value                          |
|----------------|--------------------------------|
| Size           | 134.1 MiB                     |
| Duration       | 65.1 seconds                   |
| Codec          | H.264 High (avc1)             |
| Resolution     | 1920x1080                      |
| Frame rate     | ~30 fps                        |
| Video bitrate  | ~17.0 Mbps                     |
| Audio          | AAC ~256 kbps                  |
| Source device  | Android 7.0 phone (2019)      |

Both files are H.264 phone recordings at high bitrate. The phones used
efficient hardware H.264 encoders at high quality settings.

## How the Tool Selects CRF

The tool re-encodes every file from its source codec to H.265 (libx265).
The CRF (Constant Rate Factor) controls the quality/size tradeoff: lower
CRF = higher quality = bigger file.

Each compression mode has a base CRF:

| Mode                              | Base CRF | Preset |
|-----------------------------------|----------|--------|
| Quality priority                  | 22       | slow   |
| Keep quality and reduce size      | 26       | slow   |
| Size priority                     | 30       | fast   |

A dynamic adjustment adds 0 to +6 based on the source's bitrate density
(bits per megapixel per frame). Higher density = the tool assumes more
headroom for compression and keeps the CRF low. Lower density = the tool
assumes the source is already compressed and raises the CRF.

### Effective CRF for each source file

**20210926_100615_1(short).mp4** — bitrate density norm = 6.95 — offset +4

| Mode             | Base | Offset | Effective CRF |
|------------------|------|--------|---------------|
| Quality          | 22   | +4     | 26            |
| Balanced         | 26   | +4     | 30            |
| Size             | 30   | +4     | 34            |

**20190903_140907.mp4** — bitrate density norm = 8.34 — offset +2

| Mode             | Base | Offset | Effective CRF |
|------------------|------|--------|---------------|
| Quality          | 22   | +2     | 24            |
| Balanced         | 26   | +2     | 28            |
| Size             | 30   | +2     | 32            |

The higher bitrate file (17 Mbps) gets a *smaller* offset (+2) than the
lower bitrate file (14 Mbps, +4). The algorithm reads high bitrate as
"raw footage with compression headroom" and applies less adjustment. For
phone videos, this is the wrong assumption — phones record at high
bitrate with efficient encoders.

## Client Test Results Explained

All tests on 20210926_100615_1(short).mp4 (25.9 MiB):

| Test | Mode              | Effective CRF | Output   | Change  | Outcome |
|------|-------------------|---------------|----------|---------|---------|
| 1    | Quality           | 26            | 43.0 MiB | +66%    | Deleted |
| 2    | Balanced          | 30            | 23.2 MiB | -10%    | Deleted |
| 3    | Size              | 34            | 9.0 MiB  | -65%    | Kept    |
| 4    | Balanced + 720p   | 30            | 3.3 MiB  | -87%    | Kept    |
| 5    | Balanced + 128k   | 30            | 23.0 MiB | -11%    | Deleted |
| 6    | Balanced, no skip | 30            | 23.2 MiB | -10%    | Deleted |
| 7    | Balanced + 30fps  | 30            | 23.2 MiB | -10%    | Deleted |

On 20190903_140907.mp4 (134.1 MiB):

| Test | Mode     | Effective CRF | Output    | Change | Outcome |
|------|----------|---------------|-----------|--------|---------|
| 1    | Quality  | 24            | 128.8 MiB | -4%   | Deleted |

### Why each result happened

**Quality grew the file (+66%).** H.265 CRF 26 targets roughly the same
visual quality as this H.264 source. The source phone encoder was
efficient at 14 Mbps. Asking H.265 to match or exceed that quality
required more bits, not fewer. Re-encoding a lossy source at a
high-quality target can produce a larger file because the encoder tries
to preserve compression artefacts as if they were real detail.

**Balanced saved only 10%.** H.265 CRF 30 is slightly below the source
quality. The codec efficiency advantage of H.265 over H.264 (~25-50% at
equal quality) is real but the CRF is barely below the source, so the
savings are small. This is correct behaviour, not a failure.

**Size saved 65%.** H.265 CRF 34 targets noticeably lower quality than
the source. The encoder freely discards detail, producing a much smaller
file. Visible quality loss is expected.

**720p saved 87%.** Downscaling from 1080p to 720p removes 56% of pixels.
Combined with H.265 re-encoding, savings are dramatic.

**Audio/framerate changes made no difference.** The source audio is already
252 kbps AAC. Re-encoding to 128 kbps saved ~0.2 MiB. The frame rate is
already ~30 fps so capping at 30 fps changes nothing.

## The Deletion Bug

The tool had a post-encode check: if the output was >= 80% of the input
size, it deleted the output file and reported failure. Tests 1, 2, 5, 6,
and 7 all produced valid compressed files that were then deleted.

- Test 2 saved 2.7 MiB per file. Across hundreds of files, that is real
  savings the client never received.
- Test 1 (output larger than input) was correctly rejected — a bigger
  file has no value.

**Fix applied:** Output is now only deleted when it is larger than the
input. Any reduction, however small, keeps the file. A warning is shown
when savings are below 20%.

## The Modes Question

The client asked: "What is the difference between Quality priority and
Keep quality and reduce size? Do we really need both?"

On these source files, Quality priority produced worse results than
Balanced in every test — larger files or smaller savings. The 4-CRF-point
difference between them (base 22 vs 26) only matters for raw or
poorly-compressed footage where a lower CRF preserves detail that would
otherwise be lost. For already-compressed phone videos, the extra quality
target is wasted and can make files grow.

Recommendation: two modes instead of three.

| Mode                              | CRF  | Preset | Use case                       |
|-----------------------------------|------|--------|--------------------------------|
| Keep quality and reduce size      | 26   | slow   | Preserve quality, moderate savings |
| Size priority                     | 30   | fast   | Aggressive savings, some quality loss |

## Summary

1. These source files are already-compressed H.264 phone videos at high
   bitrate. Re-encoding to H.265 at quality-preserving settings produces
   small but real savings (4-13%). This is expected, not a bug.
2. The tool was deleting those valid outputs because the savings were
   below the 80% threshold. This was the bug. It is now fixed.
3. Quality priority mode is harmful for this content — it can make files
   larger. It should be removed.
4. For significant size reduction on already-compressed content, use
   Size priority or downscale resolution.

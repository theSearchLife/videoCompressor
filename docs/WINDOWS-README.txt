Video Compressor (vc.exe) - native Windows build
==================================================

This zip is self-contained: vc.exe, ffmpeg.exe and ffprobe.exe live in the
same folder, so vc.exe finds them automatically. You do not need to install
ffmpeg separately or edit your PATH.

To use:
  1. Extract the zip somewhere convenient (e.g. C:\Tools\vc).
  2. Open PowerShell or Command Prompt in that folder.
  3. Run: .\vc.exe "C:\path\to\videos"

Or add the extracted folder to your PATH and run "vc" from anywhere.

Subcommands:
  vc <dir>             compress videos under <dir>
  vc cleanup <dir>     delete originals and rename converted outputs
  vc assess <dir>      run codec/CRF test matrix on samples
  vc --help            full flag reference

ffmpeg / ffprobe are bundled under GPL v3. See FFMPEG-LICENSE.txt.

If you prefer Docker Desktop, the Linux container image still works and is
the recommended path for users who already have Docker installed:
  docker run --rm -it -v C:\path\to\videos:/videos ghcr.io/thesearchlife/videocompressor:main /videos

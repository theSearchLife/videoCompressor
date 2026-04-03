$ErrorActionPreference = "Stop"
$IMAGE_NAME = "ghcr.io/thesearchlife/videocompressor:main"

function Show-Usage {
    Write-Host @"
vc — Video Compressor

Usage:
  vc <input-dir> [flags]                 Compress videos
  vc cleanup <input-dir> [flags]         Delete originals and rename converted outputs
  vc assess <input-dir> [flags]          Run codec/CRF test matrix

The input directory is mounted into the container. All other flags are passed
through to the vc binary inside Docker.

Examples:
  .\vc.ps1 C:\Videos --strategy quality --suffix _v1
  .\vc.ps1 C:\Videos --strategy size --suffix _v2 --skip-converted no
  .\vc.ps1 cleanup C:\Videos --suffix _v2
  .\vc.ps1 C:\Videos --dry-run

Run ".\vc.ps1 --help" for the full flag reference.
"@
}

function Die($msg) {
    Write-Error $msg
    exit 1
}

# --- Pre-flight checks ---

if ($args.Count -eq 0 -or $args[0] -in @("--help", "-h", "help")) {
    Show-Usage
    exit 0
}

$dockerPath = Get-Command docker -ErrorAction SilentlyContinue
if (-not $dockerPath) { Die "Docker is not installed. See https://docs.docker.com/get-docker/" }

docker info 2>$null | Out-Null
if ($LASTEXITCODE -ne 0) { Die "Docker is not running." }

# --- Parse the subcommand and input directory ---

$subcommand = ""
$inputDir = ""
$passArgs = @()

switch ($args[0]) {
    { $_ -in @("cleanup", "assess") } {
        $subcommand = $args[0]
        if ($args.Count -lt 2) { Die "Usage: vc $subcommand <input-dir> [flags]" }
        $inputDir = $args[1]
        if ($args.Count -gt 2) { $passArgs = $args[2..($args.Count - 1)] }
    }
    default {
        $inputDir = $args[0]
        if ($args.Count -gt 1) { $passArgs = $args[1..($args.Count - 1)] }
    }
}

# --- Validate input directory ---

$inputDir = Resolve-Path $inputDir -ErrorAction SilentlyContinue
if (-not $inputDir) { Die "Directory does not exist: $($args[0])" }
if (-not (Test-Path $inputDir -PathType Container)) { Die "Not a directory: $inputDir" }

# --- Pull image if needed ---

docker image inspect $IMAGE_NAME 2>$null | Out-Null
if ($LASTEXITCODE -ne 0) {
    Write-Host "Pulling $IMAGE_NAME..."
    docker pull $IMAGE_NAME
    if ($LASTEXITCODE -ne 0) { Die "Docker pull failed." }
    Write-Host ""
}

# --- Run ---

$dockerArgs = @("run", "--rm", "-it", "-v", "${inputDir}:/videos")

if ($subcommand) {
    docker @dockerArgs $IMAGE_NAME $subcommand /videos @passArgs
} else {
    docker @dockerArgs $IMAGE_NAME /videos @passArgs
}

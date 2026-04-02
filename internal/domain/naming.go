package domain

import (
	"fmt"
	"path/filepath"
	"strings"
)

const OutputExtension = ".mp4"

// CompressOutputPath builds the output path using the user-chosen suffix.
// The suffix includes the leading underscore, e.g. "_compressed".
func CompressOutputPath(inputPath string, suffix string) string {
	dir := filepath.Dir(inputPath)
	base := strings.TrimSuffix(filepath.Base(inputPath), filepath.Ext(inputPath))
	return filepath.Join(dir, fmt.Sprintf("%s%s%s", base, suffix, OutputExtension))
}

// SplitOutputPath checks whether path is a suffixed output file.
// Returns the original base name (without suffix) and true if it matches.
func SplitOutputPath(path string, suffix string) (string, bool) {
	if strings.ToLower(filepath.Ext(path)) != OutputExtension {
		return "", false
	}
	base := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	if strings.HasSuffix(base, suffix) {
		return base[:len(base)-len(suffix)], true
	}
	return "", false
}

// IsOutputFile returns true if the filename bears the given suffix,
// indicating it was produced by a previous compression run.
func IsOutputFile(path string, suffix string) bool {
	_, ok := SplitOutputPath(path, suffix)
	return ok
}

func BaseOutputPath(inputPath string) string {
	dir := filepath.Dir(inputPath)
	base := strings.TrimSuffix(filepath.Base(inputPath), filepath.Ext(inputPath))
	return filepath.Join(dir, base+OutputExtension)
}

func TempOutputPath(finalPath string) string {
	return finalPath + ".tmp"
}

func AssessOutputFilename(inputBase string, profile Profile, res Resolution) string {
	base := strings.TrimSuffix(inputBase, filepath.Ext(inputBase))
	return fmt.Sprintf("%s_%s_crf%d_%s_%s%s",
		base, CodecSlug(profile.Codec), profile.CRF, profile.Preset, res, OutputExtension)
}

func EffectiveResolution(sourceHeight int, target Resolution) Resolution {
	if target == "" {
		return ""
	}
	targetH := target.Height()
	if sourceHeight <= targetH {
		return heightToResolution(sourceHeight)
	}
	return target
}

func heightToResolution(h int) Resolution {
	switch {
	case h >= 2160:
		return Res4K
	case h >= 1080:
		return Res1080p
	case h > 0:
		return Res720p
	default:
		return ""
	}
}

func ScaleFilter(sourceHeight int, target Resolution) string {
	if target == "" {
		return ""
	}
	targetH := target.Height()
	if sourceHeight <= targetH {
		return ""
	}
	return fmt.Sprintf("scale=-2:%d", targetH)
}

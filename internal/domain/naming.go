package domain

import (
	"fmt"
	"path/filepath"
	"strings"
)

// resolutionSuffixes are appended to output filenames. Files already bearing one
// of these suffixes are previous outputs and should be skipped during scanning.
var resolutionSuffixes = []string{"_720p", "_1080p", "_4k"}

// IsOutputFile returns true if the filename already has a resolution suffix,
// indicating it was produced by a previous compression run.
func IsOutputFile(path string) bool {
	base := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	base = strings.ToLower(base)
	for _, s := range resolutionSuffixes {
		if strings.HasSuffix(base, s) {
			return true
		}
	}
	return false
}

func CompressOutputPath(inputPath string, effectiveRes Resolution) string {
	dir := filepath.Dir(inputPath)
	base := strings.TrimSuffix(filepath.Base(inputPath), filepath.Ext(inputPath))
	return filepath.Join(dir, fmt.Sprintf("%s_%s.mp4", base, effectiveRes))
}

func AssessOutputFilename(inputBase string, profile Profile, res Resolution) string {
	base := strings.TrimSuffix(inputBase, filepath.Ext(inputBase))
	return fmt.Sprintf("%s_%s_crf%d_%s_%s.mp4",
		base, CodecSlug(profile.Codec), profile.CRF, profile.Preset, res)
}

func EffectiveResolution(sourceHeight int, target Resolution) Resolution {
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
	targetH := target.Height()
	if sourceHeight <= targetH {
		return ""
	}
	return fmt.Sprintf("scale=-2:%d", targetH)
}

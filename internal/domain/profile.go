package domain

var StrategyProfiles = map[CompressionStrategy]Profile{
	StrategyQuality: {
		Name:            "quality",
		Codec:           "libx265",
		CRF:             22,
		Preset:          "slow",
		AudioCodec:      "aac",
		AudioBitrate:    "128k",
		ContainerFormat: "mp4",
	},
	StrategyBalanced: {
		Name:            "balanced",
		Codec:           "libx265",
		CRF:             26,
		Preset:          "slow",
		AudioCodec:      "aac",
		AudioBitrate:    "128k",
		ContainerFormat: "mp4",
	},
	StrategySizePriority: {
		Name:            "size",
		Codec:           "libx265",
		CRF:             30,
		Preset:          "fast",
		AudioCodec:      "aac",
		AudioBitrate:    "96k",
		ContainerFormat: "mp4",
	},
}

// SelectCRF returns a dynamic CRF based on source properties.
// It normalises the source bitrate per megapixel at 30fps and offsets
// the base CRF so that already-efficient sources get a higher CRF.
func SelectCRF(strategy CompressionStrategy, source VideoMeta) int {
	base := StrategyProfiles[strategy].CRF

	norm, ok := normalizedBitrate(source)
	if !ok {
		return base
	}

	switch {
	case norm > 15:
		return base // raw/prosumer footage — base CRF is fine
	case norm > 8:
		return base + 2 // DSLR / high-bitrate phone
	case norm > 4:
		return base + 4 // moderate — phone/camera with decent encoding
	default:
		return base + 6 // already well-compressed — streaming rips, etc.
	}
}

type CompressionAdvice struct {
	Skip    bool
	Message string
}

// AssessCompression returns guidance for sources that are unlikely to shrink
// meaningfully unless the user also lowers quality, resolution, frame rate, or audio bitrate.
func AssessCompression(strategy CompressionStrategy, source VideoMeta, profile Profile, target Resolution) CompressionAdvice {
	norm, ok := normalizedBitrate(source)
	if !ok {
		return CompressionAdvice{}
	}

	downscale := target != "" && source.Height > target.Height()
	fpsReduce := profile.FrameRate > 0 && source.FrameRate > 0 && float64(profile.FrameRate) < source.FrameRate
	audioReduce := profile.AudioCodec != "" && profile.AudioCodec != "copy"

	if downscale || fpsReduce || audioReduce || strategy == StrategySizePriority {
		return CompressionAdvice{}
	}

	if source.Codec == "hevc" && norm <= 8 {
		return CompressionAdvice{
			Skip:    true,
			Message: "already HEVC and efficiently compressed at original settings; use Size (fast) or reduce resolution/fps/audio for meaningful savings",
		}
	}

	if strategy == StrategyQuality && norm <= 8 {
		return CompressionAdvice{
			Skip:    true,
			Message: "already compressed source with limited headroom; Quality (slow) may increase size. Use Balanced, Size (fast), or reduce resolution/fps/audio",
		}
	}

	if strategy == StrategyBalanced && norm <= 4 {
		return CompressionAdvice{
			Skip:    true,
			Message: "already efficiently compressed at original settings; Balanced may save very little. Use Size (fast) or reduce resolution/fps/audio",
		}
	}

	return CompressionAdvice{}
}

func ApplyAudioMode(p Profile, mode AudioMode) Profile {
	switch mode {
	case AudioKeep:
		p.AudioCodec = "copy"
		p.AudioBitrate = ""
	case AudioLow:
		p.AudioCodec = "aac"
		p.AudioBitrate = "96k"
	case AudioMedium:
		p.AudioCodec = "aac"
		p.AudioBitrate = "128k"
	case AudioHigh:
		p.AudioCodec = "aac"
		p.AudioBitrate = "192k"
	}
	return p
}

func normalizedBitrate(source VideoMeta) (float64, bool) {
	if source.Duration == 0 || source.Width == 0 || source.Height == 0 {
		return 0, false
	}

	sourceBitrateMbps := float64(source.Size) * 8 / source.Duration.Seconds() / 1e6
	mpx := float64(source.Width) * float64(source.Height) / 1e6
	if mpx == 0 {
		return 0, false
	}

	fps := source.FrameRate
	if fps == 0 {
		fps = 30
	}

	// Normalised bitrate: Mbps per megapixel, scaled to 30fps reference.
	// Higher values = more headroom for compression.
	return sourceBitrateMbps / mpx * (30 / fps), true
}

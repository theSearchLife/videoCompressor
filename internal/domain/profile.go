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

	if source.Duration == 0 || source.Width == 0 || source.Height == 0 {
		return base
	}

	sourceBitrateMbps := float64(source.Size) * 8 / source.Duration.Seconds() / 1e6
	mpx := float64(source.Width) * float64(source.Height) / 1e6

	fps := source.FrameRate
	if fps == 0 {
		fps = 30
	}

	// Normalised bitrate: Mbps per megapixel, scaled to 30fps reference.
	// Higher values = more headroom for compression.
	norm := sourceBitrateMbps / mpx * (30 / fps)

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

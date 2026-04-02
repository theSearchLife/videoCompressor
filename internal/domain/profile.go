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
		CRF:             28,
		Preset:          "fast",
		AudioCodec:      "aac",
		AudioBitrate:    "96k",
		ContainerFormat: "mp4",
	},
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

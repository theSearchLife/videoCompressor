package domain

var BuiltInProfiles = map[CompressionLevel]Profile{
	CompressionHigh: {
		Name:            "high",
		Codec:           "libx265",
		CRF:             26,
		Preset:          "slow",
		AudioCodec:      "aac",
		AudioBitrate:    "128k",
		ContainerFormat: "mp4",
	},
	CompressionLow: {
		Name:            "low",
		Codec:           "libx265",
		CRF:             28,
		Preset:          "fast",
		AudioCodec:      "aac",
		AudioBitrate:    "128k",
		ContainerFormat: "mp4",
	},
}

package domain

import "strings"

func CodecSlug(codec string) string {
	switch codec {
	case "libx265":
		return "h265"
	case "libx264":
		return "h264"
	default:
		return strings.TrimPrefix(codec, "lib")
	}
}

func CodecDisplayName(codec string) string {
	switch codec {
	case "libx265":
		return "H.265"
	case "libx264":
		return "H.264"
	default:
		return strings.TrimPrefix(codec, "lib")
	}
}

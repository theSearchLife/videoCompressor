package domain

import "strings"

// mp4CompatibleAudioCodecs lists the audio codecs that the MP4 muxer accepts
// without a transcode. Sources outside this set (notably PCM variants from
// older camcorders) must be re-encoded to AAC or the mux will fail with
// "Could not find tag for codec ... in stream".
var mp4CompatibleAudioCodecs = map[string]bool{
	"aac": true, "ac3": true, "eac3": true, "mp3": true,
	"alac": true, "mp2": true, "mp1": true, "opus": true,
	"flac": true, "amr_nb": true, "amr_wb": true,
}

func IsMP4CompatibleAudioCodec(codec string) bool {
	return mp4CompatibleAudioCodecs[strings.ToLower(strings.TrimSpace(codec))]
}

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

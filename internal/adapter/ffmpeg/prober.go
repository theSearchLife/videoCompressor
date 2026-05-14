package ffmpeg

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/theSearchLife/videoCompressor/internal/domain"
)

type Prober struct{}

func NewProber() *Prober {
	return &Prober{}
}

type probeOutput struct {
	Streams []probeStream `json:"streams"`
	Format  probeFormat   `json:"format"`
}

type probeStream struct {
	CodecType        string            `json:"codec_type"`
	Width            int               `json:"width"`
	Height           int               `json:"height"`
	CodecName        string            `json:"codec_name"`
	AvgFrameRate     string            `json:"avg_frame_rate"`
	PixelFormat      string            `json:"pix_fmt"`
	BitsPerRawSample string            `json:"bits_per_raw_sample"`
	ColorRange       string            `json:"color_range"`
	ColorSpace       string            `json:"color_space"`
	ColorTransfer    string            `json:"color_transfer"`
	ColorPrimaries   string            `json:"color_primaries"`
	Tags             map[string]string `json:"tags"`
}

type probeFormat struct {
	Duration string            `json:"duration"`
	Size     string            `json:"size"`
	Tags     map[string]string `json:"tags"`
}

func (p *Prober) Probe(ctx context.Context, path string) (domain.VideoMeta, error) {
	cmd := exec.CommandContext(ctx, "ffprobe",
		"-v", "quiet",
		"-print_format", "json",
		"-show_format",
		"-show_streams",
		path,
	)

	out, err := cmd.Output()
	if err != nil {
		return domain.VideoMeta{}, fmt.Errorf("ffprobe %s: %w", path, err)
	}

	var probe probeOutput
	if err := json.Unmarshal(out, &probe); err != nil {
		return domain.VideoMeta{}, fmt.Errorf("parse ffprobe output: %w", err)
	}
	ffprobeSLog3 := detectSLog3FromJSON(out)

	var video, audio *probeStream
	for i := range probe.Streams {
		s := &probe.Streams[i]
		if video == nil && (s.CodecType == "video" || (s.CodecType == "" && s.Width > 0)) {
			video = s
		}
		if audio == nil && s.CodecType == "audio" {
			audio = s
		}
	}
	if video == nil {
		return domain.VideoMeta{}, fmt.Errorf("no video stream found in %s", path)
	}

	dur, _ := strconv.ParseFloat(probe.Format.Duration, 64)
	size, _ := strconv.ParseInt(probe.Format.Size, 10, 64)

	audioCodec := ""
	if audio != nil {
		audioCodec = audio.CodecName
	}

	mediaInfoSLog3, mediaInfoSource := detectSLog3WithMediaInfo(ctx, path)
	slog3 := ffprobeSLog3 || mediaInfoSLog3
	slog3Detection := ""
	switch {
	case ffprobeSLog3 && mediaInfoSLog3:
		slog3Detection = "ffprobe+mediainfo"
	case ffprobeSLog3:
		slog3Detection = "ffprobe"
	case mediaInfoSLog3:
		slog3Detection = mediaInfoSource
	}

	return domain.VideoMeta{
		Path:           path,
		Width:          video.Width,
		Height:         video.Height,
		Duration:       time.Duration(dur * float64(time.Second)),
		Codec:          video.CodecName,
		AudioCodec:     audioCodec,
		Size:           size,
		FrameRate:      parseFrameRate(video.AvgFrameRate),
		PixelFormat:    video.PixelFormat,
		BitDepth:       bitDepth(video.BitsPerRawSample, video.PixelFormat),
		SLog3:          slog3,
		SLog3Detection: slog3Detection,
	}, nil
}

func detectSLog3WithMediaInfo(ctx context.Context, path string) (bool, string) {
	cmd := exec.CommandContext(ctx, "mediainfo", "--Output=JSON", path)
	out, err := cmd.Output()
	if err != nil {
		return false, ""
	}
	if detectSLog3FromJSON(out) {
		return true, "mediainfo"
	}
	return false, ""
}

func detectSLog3FromJSON(out []byte) bool {
	var data any
	if err := json.Unmarshal(out, &data); err != nil {
		return false
	}
	return metadataContainsSLog3("", data)
}

func metadataContainsSLog3(key string, value any) bool {
	switch v := value.(type) {
	case map[string]any:
		for childKey, childValue := range v {
			if metadataContainsSLog3(childKey, childValue) {
				return true
			}
		}
	case []any:
		for _, childValue := range v {
			if metadataContainsSLog3(key, childValue) {
				return true
			}
		}
	case string:
		return isColourMetadataKey(key) && isSLog3Value(v)
	}
	return false
}

func isColourMetadataKey(key string) bool {
	k := strings.ToLower(key)
	if k == "" || k == "@ref" || strings.Contains(k, "filename") || strings.Contains(k, "file_") || strings.Contains(k, "extension") {
		return false
	}
	return strings.Contains(k, "transfer") ||
		strings.Contains(k, "gamma") ||
		strings.Contains(k, "colour") ||
		strings.Contains(k, "color") ||
		strings.Contains(k, "primaries") ||
		strings.Contains(k, "matrix") ||
		strings.Contains(k, "profile")
}

func isSLog3Value(value string) bool {
	normalised := strings.ToLower(value)
	replacer := strings.NewReplacer("-", "", "_", "", " ", "", ".", "")
	if strings.Contains(replacer.Replace(normalised), "slog3") {
		return true
	}

	hex := strings.Builder{}
	for _, r := range normalised {
		if (r >= '0' && r <= '9') || (r >= 'a' && r <= 'f') {
			hex.WriteRune(r)
		}
	}
	h := hex.String()
	return strings.Contains(h, "0e06040101010604") ||
		strings.Contains(h, "0e06040101010605") ||
		strings.Contains(h, "060e2b34040101060e06040101010604") ||
		strings.Contains(h, "060e2b34040101060e06040101010605")
}

func bitDepth(bitsPerRawSample, pixFmt string) int {
	if bitsPerRawSample != "" {
		if depth, err := strconv.Atoi(bitsPerRawSample); err == nil {
			return depth
		}
	}
	for _, depth := range []int{16, 14, 12, 10, 9} {
		if strings.Contains(pixFmt, strconv.Itoa(depth)) {
			return depth
		}
	}
	if pixFmt != "" {
		return 8
	}
	return 0
}

func parseFrameRate(s string) float64 {
	parts := strings.SplitN(s, "/", 2)
	if len(parts) != 2 {
		f, _ := strconv.ParseFloat(s, 64)
		return f
	}
	num, _ := strconv.ParseFloat(parts[0], 64)
	den, _ := strconv.ParseFloat(parts[1], 64)
	if den == 0 {
		return 0
	}
	return num / den
}

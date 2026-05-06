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
	CodecType    string `json:"codec_type"`
	Width        int    `json:"width"`
	Height       int    `json:"height"`
	CodecName    string `json:"codec_name"`
	AvgFrameRate string `json:"avg_frame_rate"`
}

type probeFormat struct {
	Duration string `json:"duration"`
	Size     string `json:"size"`
}

func (p *Prober) Probe(ctx context.Context, path string) (domain.VideoMeta, error) {
	cmd := exec.CommandContext(ctx, resolveBinary("ffprobe"),
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

	return domain.VideoMeta{
		Path:       path,
		Width:      video.Width,
		Height:     video.Height,
		Duration:   time.Duration(dur * float64(time.Second)),
		Codec:      video.CodecName,
		AudioCodec: audioCodec,
		Size:       size,
		FrameRate:  parseFrameRate(video.AvgFrameRate),
	}, nil
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

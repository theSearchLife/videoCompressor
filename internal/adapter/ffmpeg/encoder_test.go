package ffmpeg

import (
	"testing"
	"time"

	"github.com/theSearchLife/videoCompressor/internal/domain"
)

func TestBuildArgsForCopyAudioAndScaling(t *testing.T) {
	job := domain.Job{
		Input: domain.VideoMeta{
			Path:     "/videos/source.mp4",
			Height:   2160,
			Duration: 10 * time.Second,
		},
		OutputPath: "/videos/source_1080p.mp4",
		Profile: domain.Profile{
			Codec:           "libx265",
			CRF:             26,
			Preset:          "slow",
			AudioCodec:      "copy",
			ContainerFormat: "mp4",
		},
		Resolution: domain.Res1080p,
	}

	args := buildArgs(job)
	assertContains(t, args, "-vf")
	assertContains(t, args, "scale=-2:1080")
	assertContains(t, args, "-c:a")
	assertContains(t, args, "copy")
}

func TestParseProgress(t *testing.T) {
	progress, ok := parseProgress("out_time_us=5000000", 10*time.Second)
	if !ok {
		t.Fatal("expected progress to parse")
	}
	if progress != 0.5 {
		t.Fatalf("expected 0.5 progress, got %v", progress)
	}
}

func assertContains(t *testing.T, values []string, needle string) {
	t.Helper()
	for _, value := range values {
		if value == needle {
			return
		}
	}
	t.Fatalf("expected %q in %v", needle, values)
}

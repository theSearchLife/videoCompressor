package ffmpeg

import (
	"context"
	"os"
	"path/filepath"
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
	assertContains(t, args, "/videos/source_1080p.mp4.tmp")
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

func TestEncodeRenamesTmpOutputOnSuccess(t *testing.T) {
	binDir := t.TempDir()
	ffmpegPath := filepath.Join(binDir, "ffmpeg")
	script := "#!/bin/sh\nfor last; do :; done\noutfile=\"$last\"\nprintf 'out_time_us=1000000\\nprogress=end\\n'\nprintf 'encoded' > \"$outfile\"\n"
	if err := os.WriteFile(ffmpegPath, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}

	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	outputDir := t.TempDir()
	finalOutput := filepath.Join(outputDir, "source_1080p.mp4")
	job := domain.Job{
		Input: domain.VideoMeta{
			Path:     filepath.Join(outputDir, "source.mp4"),
			Height:   2160,
			Duration: time.Second,
		},
		OutputPath: finalOutput,
		Profile: domain.Profile{
			Codec:           "libx265",
			CRF:             26,
			Preset:          "slow",
			AudioCodec:      "copy",
			ContainerFormat: "mp4",
		},
		Resolution: domain.Res1080p,
	}

	if err := os.WriteFile(job.Input.Path, []byte("source"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := NewEncoder().Encode(context.Background(), job, nil); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(finalOutput); err != nil {
		t.Fatalf("expected final output to exist: %v", err)
	}
	if _, err := os.Stat(domain.TempOutputPath(finalOutput)); !os.IsNotExist(err) {
		t.Fatalf("expected tmp output to be renamed away, got err=%v", err)
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

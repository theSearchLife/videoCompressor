package adapter

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/theSearchLife/videoCompressor/internal/domain"
)

func TestFormatLiveLineCompactsFilenameToFitWidth(t *testing.T) {
	job := domain.Job{
		ID:         3,
		OutputPath: "/videos/a-very-long-client-filename-that-would-wrap-in-a-normal-terminal_compressed.mp4",
	}

	got := formatLiveLine(job, 0.42, 2*time.Minute, 3*time.Minute, 76)

	if width := textWidth(got); width > 76 {
		t.Fatalf("expected line width <= 76, got %d for %q", width, got)
	}
	if !strings.Contains(got, "elapsed 02:00") {
		t.Fatalf("expected elapsed time to be preserved, got %q", got)
	}
	if !strings.Contains(got, "eta 03:00") {
		t.Fatalf("expected eta to be preserved, got %q", got)
	}
	if !strings.Contains(got, "...") {
		t.Fatalf("expected filename to be compacted, got %q", got)
	}
	if !strings.Contains(got, ".mp4") {
		t.Fatalf("expected filename extension to be preserved, got %q", got)
	}
}

func TestLiveProgressDrawsOneTerminalRowPerLine(t *testing.T) {
	var buf bytes.Buffer
	progress := &liveProgress{
		out: &buf,
		tty: true,
		terminalWidth: func() int {
			return 32
		},
	}

	progress.addLine(1, strings.Repeat("x", 100))

	got := strings.TrimSuffix(buf.String(), "\n")
	if strings.Contains(got, "\n") {
		t.Fatalf("expected one rendered line, got %q", got)
	}
	if width := textWidth(got); width > 31 {
		t.Fatalf("expected rendered width <= 31, got %d for %q", width, got)
	}
	if progress.drawn != 1 {
		t.Fatalf("expected drawn row count 1, got %d", progress.drawn)
	}
}

func TestFitLiveTextRemovesEmbeddedLineBreaks(t *testing.T) {
	got := fitLiveText("first\nsecond\rthird", 80)

	if strings.ContainsAny(got, "\n\r") {
		t.Fatalf("expected line breaks to be removed, got %q", got)
	}
}

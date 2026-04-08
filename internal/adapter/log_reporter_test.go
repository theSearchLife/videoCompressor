package adapter

import (
	"bytes"
	"errors"
	"log"
	"strings"
	"testing"

	"github.com/theSearchLife/videoCompressor/internal/domain"
)

func TestLogReporterJobFinishedFailureIncludesSizes(t *testing.T) {
	var buf bytes.Buffer
	prevWriter := log.Writer()
	prevFlags := log.Flags()
	log.SetOutput(&buf)
	log.SetFlags(0)
	defer log.SetOutput(prevWriter)
	defer log.SetFlags(prevFlags)

	reporter := NewLogReporter()
	job := domain.Job{
		ID: 7,
		Input: domain.VideoMeta{
			Path: "input.mp4",
		},
	}
	result := domain.Result{
		InputSize:  194171921,
		OutputSize: 220000000,
		Error:      errors.New("output larger than input (13% increase): try the size profile for this file"),
	}

	reporter.JobFinished(job, result)

	got := buf.String()
	if !strings.Contains(got, "185.2 MB") {
		t.Fatalf("expected input size in log output, got %q", got)
	}
	if !strings.Contains(got, "209.8 MB") {
		t.Fatalf("expected output size in log output, got %q", got)
	}
	if !strings.Contains(got, "FAIL: input.mp4") {
		t.Fatalf("expected failure log to include file name, got %q", got)
	}
}

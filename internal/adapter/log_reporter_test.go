package adapter

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/theSearchLife/videoCompressor/internal/domain"
)

func TestLogReporterJobFinishedFailureIncludesSizes(t *testing.T) {
	var buf bytes.Buffer
	reporter := newLogReporter(&buf)

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

func TestLogReporterSummaryListsFailures(t *testing.T) {
	var buf bytes.Buffer
	reporter := newLogReporter(&buf)

	results := []domain.Result{
		{
			Job: domain.Job{Input: domain.VideoMeta{Path: "/v/ok.mp4"}},
		},
		{
			Job:   domain.Job{Input: domain.VideoMeta{Path: "/v/big.mp4"}},
			Error: errors.New("output larger than input (5% increase): try the size profile for this file"),
		},
		{
			Job:   domain.Job{Input: domain.VideoMeta{Path: "/v/broken.mov"}},
			Error: errors.New("ffmpeg encode: codec not supported"),
		},
	}

	reporter.Summary(results, 1)

	got := buf.String()
	if !strings.Contains(got, "Failed files (2)") {
		t.Fatalf("expected failure header in summary, got %q", got)
	}
	if !strings.Contains(got, "big.mp4: output larger than input") {
		t.Fatalf("expected size-failure listing, got %q", got)
	}
	if !strings.Contains(got, "broken.mov: ffmpeg encode: codec not supported") {
		t.Fatalf("expected encode-failure listing, got %q", got)
	}
}

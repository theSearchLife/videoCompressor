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
	if !strings.Contains(got, "[7]") {
		t.Fatalf("expected row id [7] in failure row, got %q", got)
	}
	if !strings.Contains(got, "FAIL input.mp4") {
		t.Fatalf("expected unified FAIL row to include file name, got %q", got)
	}
	if !strings.Contains(got, "failed") {
		t.Fatalf("expected slot label 'failed' in FAIL row, got %q", got)
	}
}

func TestLogReporterJobFinishedDoneRowFormat(t *testing.T) {
	var buf bytes.Buffer
	reporter := newLogReporter(&buf)

	job := domain.Job{
		ID:         3,
		Input:      domain.VideoMeta{Path: "/videos/clip.mov"},
		OutputPath: "/videos/clip_compressed.mp4",
	}
	result := domain.Result{
		InputSize:  9_500_000,
		OutputSize: 4_000_000,
		EncodeTime: 81 * 1_000_000_000,
	}

	reporter.JobFinished(job, result)

	got := buf.String()
	wantBits := []string{
		"[3]",
		"[####################]",
		"DONE clip.mov",
		"9.1 MB -> 3.8 MB",
		"reduction",
	}
	for _, want := range wantBits {
		if !strings.Contains(got, want) {
			t.Fatalf("expected DONE row to contain %q, got %q", want, got)
		}
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

	skips := []domain.SkipInfo{
		{Path: "/v/skipme.mov", Size: 5_000_000, Reason: "output already exists"},
	}
	reporter.Summary(results, skips)

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
	if !strings.Contains(got, "Skipped files (1)") {
		t.Fatalf("expected skipped header in summary, got %q", got)
	}
	if !strings.Contains(got, "skipme.mov: output already exists") {
		t.Fatalf("expected skip listing in summary, got %q", got)
	}
	if !strings.Contains(got, "1 skipped") {
		t.Fatalf("expected skipped count in summary line, got %q", got)
	}
}

func TestLogReporterFileSkippedFormat(t *testing.T) {
	var buf bytes.Buffer
	reporter := newLogReporter(&buf)

	reporter.FileSkipped(domain.SkipInfo{
		RowID:  9,
		Path:   "/videos/clip.mov",
		Size:   12_500_000,
		Code:   domain.SkipCodeUncompressed,
		Reason: "already HEVC and efficiently compressed at original settings",
	})

	got := buf.String()
	wantBits := []string{
		"[9]",
		"uncompreseable",
		"SKIPED clip.mov",
		"11.9 MB",
		"already HEVC and efficiently compressed at original settings",
	}
	for _, want := range wantBits {
		if !strings.Contains(got, want) {
			t.Fatalf("expected SKIPED row to contain %q, got %q", want, got)
		}
	}
}

func TestLogReporterFileSkippedLabelsByCode(t *testing.T) {
	cases := []struct {
		code  domain.SkipCode
		label string
	}{
		{domain.SkipCodePrevCompress, "prev_compress"},
		{domain.SkipCodeAlreadyDone, "already_done"},
		{domain.SkipCodePathCollision, "path_collision"},
		{domain.SkipCodeUncompressed, "uncompreseable"},
	}
	for _, tc := range cases {
		var buf bytes.Buffer
		reporter := newLogReporter(&buf)
		reporter.FileSkipped(domain.SkipInfo{
			RowID:  0,
			Path:   "/videos/clip.mov",
			Size:   1024,
			Code:   tc.code,
			Reason: "test",
		})
		got := buf.String()
		if !strings.Contains(got, tc.label) {
			t.Fatalf("code %q: expected slot label %q in row, got %q", tc.code, tc.label, got)
		}
	}
}

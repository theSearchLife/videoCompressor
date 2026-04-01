package app

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/theSearchLife/videoCompressor/internal/domain"
)

type fakeScanner struct {
	files []string
}

func (f fakeScanner) Scan(context.Context, string, bool) ([]string, error) {
	return f.files, nil
}

type fakeProber struct {
	meta domain.VideoMeta
}

func (f fakeProber) Probe(context.Context, string) (domain.VideoMeta, error) {
	return f.meta, nil
}

type fakeEncoder struct{}

func (fakeEncoder) Encode(_ context.Context, job domain.Job, onProgress func(float64)) error {
	if onProgress != nil {
		onProgress(1)
	}
	return os.WriteFile(job.OutputPath, []byte("encoded"), 0o644)
}

type fakeReporter struct{}

func (fakeReporter) JobStarted(domain.Job)           {}
func (fakeReporter) JobProgress(domain.Job, float64) {}
func (fakeReporter) JobFinished(domain.Job, error)   {}
func (fakeReporter) Summary([]domain.Result)         {}

func TestAssessorWritesReportArtifacts(t *testing.T) {
	inputPath := "/samples/client.mp4"
	outputDir := t.TempDir()

	assessor := NewAssessor(
		fakeScanner{files: []string{inputPath}},
		fakeProber{meta: domain.VideoMeta{
			Path:     inputPath,
			Width:    1920,
			Height:   1080,
			Duration: 5 * time.Second,
			Codec:    "h264",
			Size:     1024,
		}},
		fakeEncoder{},
		fakeReporter{},
		nil,
	)

	err := assessor.Run(context.Background(), AssessOptions{
		InputDir:  "/samples",
		OutputDir: outputDir,
		Matrix: domain.MatrixConfig{
			Codecs:      []string{"libx265"},
			CRFs:        map[string][]int{"libx265": {26}},
			Presets:     []string{"slow"},
			Resolutions: []domain.Resolution{domain.Res720p},
		},
		Workers: 2,
	})
	if err != nil {
		t.Fatal(err)
	}

	matches, err := filepath.Glob(filepath.Join(outputDir, "*", "report.md"))
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 1 {
		t.Fatalf("expected 1 report file, got %d", len(matches))
	}

	csvMatches, err := filepath.Glob(filepath.Join(outputDir, "*", "results.csv"))
	if err != nil {
		t.Fatal(err)
	}
	if len(csvMatches) != 1 {
		t.Fatalf("expected 1 csv file, got %d", len(csvMatches))
	}

	reportBytes, err := os.ReadFile(matches[0])
	if err != nil {
		t.Fatal(err)
	}
	reportText := string(reportBytes)
	if !strings.Contains(reportText, "client.mp4") {
		t.Fatalf("expected source file in report, got %q", reportText)
	}
	if !strings.Contains(reportText, "H.265") {
		t.Fatalf("expected codec label in report, got %q", reportText)
	}
}

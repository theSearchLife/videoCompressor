package app

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/theSearchLife/videoCompressor/internal/domain"
)

type stubEncoder struct {
	outputData []byte
}

func (s stubEncoder) Encode(_ context.Context, job domain.Job, onProgress func(float64)) error {
	if onProgress != nil {
		onProgress(1)
	}
	return os.WriteFile(job.OutputPath, s.outputData, 0o644)
}

type stubReporter struct {
	results []domain.Result
}

func (s *stubReporter) JobStarted(domain.Job)                     {}
func (s *stubReporter) JobProgress(domain.Job, float64)           {}
func (s *stubReporter) JobFinished(_ domain.Job, r domain.Result) { s.results = append(s.results, r) }
func (s *stubReporter) Summary([]domain.Result)                   {}

func TestOrchestratorDeletesOutputLargerThanInput(t *testing.T) {
	root := t.TempDir()
	inputPath := filepath.Join(root, "source.mp4")
	outputPath := filepath.Join(root, "source_compressed.mp4")

	if err := os.WriteFile(inputPath, make([]byte, 100), 0o644); err != nil {
		t.Fatal(err)
	}

	// Encoder writes 200 bytes (larger than input)
	reporter := &stubReporter{}
	orch := NewOrchestrator(stubEncoder{outputData: make([]byte, 200)}, reporter, 1)

	jobs := []domain.Job{{
		ID:         0,
		Input:      domain.VideoMeta{Path: inputPath, Size: 100, Duration: time.Second},
		OutputPath: outputPath,
		Profile:    domain.StrategyProfiles[domain.StrategyBalanced],
	}}

	results := orch.Run(context.Background(), jobs)

	if results[0].Error == nil {
		t.Fatal("expected error when output is larger than input")
	}
	if _, err := os.Stat(outputPath); !os.IsNotExist(err) {
		t.Fatal("expected output file to be deleted when larger than input")
	}
}

func TestOrchestratorKeepsMinimalReductionWhenOutputIsSmallerThanInput(t *testing.T) {
	root := t.TempDir()
	inputPath := filepath.Join(root, "source.mp4")
	outputPath := filepath.Join(root, "source_compressed.mp4")

	if err := os.WriteFile(inputPath, make([]byte, 1000), 0o644); err != nil {
		t.Fatal(err)
	}

	// Encoder writes 850 bytes. This is a small reduction, but still a valid
	// smaller output and should be kept.
	reporter := &stubReporter{}
	orch := NewOrchestrator(stubEncoder{outputData: make([]byte, 850)}, reporter, 1)

	jobs := []domain.Job{{
		ID:         0,
		Input:      domain.VideoMeta{Path: inputPath, Size: 1000, Duration: time.Second},
		OutputPath: outputPath,
		Profile:    domain.StrategyProfiles[domain.StrategyBalanced],
	}}

	results := orch.Run(context.Background(), jobs)

	if results[0].Error != nil {
		t.Fatalf("unexpected error: %v", results[0].Error)
	}
	if _, err := os.Stat(outputPath); err != nil {
		t.Fatal("expected smaller output to be kept")
	}
}

func TestOrchestratorKeepsOutputSmallerThanInput(t *testing.T) {
	root := t.TempDir()
	inputPath := filepath.Join(root, "source.mp4")
	outputPath := filepath.Join(root, "source_compressed.mp4")

	if err := os.WriteFile(inputPath, make([]byte, 1000), 0o644); err != nil {
		t.Fatal(err)
	}

	reporter := &stubReporter{}
	orch := NewOrchestrator(stubEncoder{outputData: make([]byte, 700)}, reporter, 1)

	jobs := []domain.Job{{
		ID:         0,
		Input:      domain.VideoMeta{Path: inputPath, Size: 1000, Duration: time.Second},
		OutputPath: outputPath,
		Profile:    domain.StrategyProfiles[domain.StrategyBalanced],
	}}

	results := orch.Run(context.Background(), jobs)

	if results[0].Error != nil {
		t.Fatalf("unexpected error: %v", results[0].Error)
	}
	if _, err := os.Stat(outputPath); err != nil {
		t.Fatal("expected output file to exist when smaller than input")
	}
}

func TestOrchestratorKeepsOutputWithMinimalReduction(t *testing.T) {
	root := t.TempDir()
	inputPath := filepath.Join(root, "source.mp4")
	outputPath := filepath.Join(root, "source_compressed.mp4")

	if err := os.WriteFile(inputPath, make([]byte, 1000), 0o644); err != nil {
		t.Fatal(err)
	}

	// Encoder writes 950 bytes (only 5% reduction, still kept).
	reporter := &stubReporter{}
	orch := NewOrchestrator(stubEncoder{outputData: make([]byte, 950)}, reporter, 1)

	jobs := []domain.Job{{
		ID:         0,
		Input:      domain.VideoMeta{Path: inputPath, Size: 1000, Duration: time.Second},
		OutputPath: outputPath,
		Profile:    domain.StrategyProfiles[domain.StrategyBalanced],
	}}

	results := orch.Run(context.Background(), jobs)

	if results[0].Error != nil {
		t.Fatalf("unexpected error: %v", results[0].Error)
	}
	if _, err := os.Stat(outputPath); err != nil {
		t.Fatal("expected output file to be kept even with minimal reduction")
	}
}

func TestOrchestratorReportsPerFileSizes(t *testing.T) {
	root := t.TempDir()
	inputPath := filepath.Join(root, "source.mp4")
	outputPath := filepath.Join(root, "source_compressed.mp4")

	if err := os.WriteFile(inputPath, make([]byte, 1000), 0o644); err != nil {
		t.Fatal(err)
	}

	reporter := &stubReporter{}
	orch := NewOrchestrator(stubEncoder{outputData: make([]byte, 500)}, reporter, 1)

	jobs := []domain.Job{{
		ID:         0,
		Input:      domain.VideoMeta{Path: inputPath, Size: 1000, Duration: time.Second},
		OutputPath: outputPath,
		Profile:    domain.StrategyProfiles[domain.StrategyBalanced],
	}}

	results := orch.Run(context.Background(), jobs)

	if results[0].InputSize != 1000 {
		t.Fatalf("expected input size 1000, got %d", results[0].InputSize)
	}
	if results[0].OutputSize != 500 {
		t.Fatalf("expected output size 500, got %d", results[0].OutputSize)
	}
	if results[0].Reduction() != 0.5 {
		t.Fatalf("expected 50%% reduction, got %.1f%%", results[0].Reduction()*100)
	}
	if len(reporter.results) != 1 {
		t.Fatalf("expected reporter to receive 1 result, got %d", len(reporter.results))
	}
}

func TestBuildJobsSkipsConvertedOutputsAndExistingFinals(t *testing.T) {
	root := t.TempDir()
	original := filepath.Join(root, "clip.mov")
	converted := filepath.Join(root, "clip_compressed.mp4")

	if err := os.WriteFile(original, []byte("source"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(converted, []byte("encoded"), 0o644); err != nil {
		t.Fatal(err)
	}

	profile := domain.StrategyProfiles[domain.StrategyBalanced]
	jobs := BuildJobs([]domain.VideoMeta{
		{Path: original, Height: 1080, Duration: time.Second},
		{Path: converted, Height: 1080, Duration: time.Second},
	}, domain.StrategyBalanced, profile, domain.Res1080p, "_compressed", true)

	if len(jobs) != 0 {
		t.Fatalf("expected no jobs when final output already exists, got %d", len(jobs))
	}
}

func TestBuildJobsDoesNotTreatOriginalNamesWithSuffixLikePatternAsOutputs(t *testing.T) {
	root := t.TempDir()
	original := filepath.Join(root, "holiday_1080p.mov")
	if err := os.WriteFile(original, []byte("source"), 0o644); err != nil {
		t.Fatal(err)
	}

	profile := domain.StrategyProfiles[domain.StrategyBalanced]
	jobs := BuildJobs([]domain.VideoMeta{
		{Path: original, Height: 1080, Duration: time.Second},
	}, domain.StrategyBalanced, profile, domain.Res720p, "_compressed", true)

	if len(jobs) != 1 {
		t.Fatalf("expected one job for an original file with a suffix-like name, got %d", len(jobs))
	}
	if got, want := jobs[0].OutputPath, filepath.Join(root, "holiday_1080p_compressed.mp4"); got != want {
		t.Fatalf("expected output %q, got %q", want, got)
	}
}

func TestBuildJobsRemovesStaleTmpForThePlannedOutputPath(t *testing.T) {
	root := t.TempDir()
	original := filepath.Join(root, "clip.mov")
	finalOutput := filepath.Join(root, "clip_compressed.mp4")
	tmpOutput := domain.TempOutputPath(finalOutput)

	if err := os.WriteFile(original, []byte("source"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(tmpOutput, []byte("partial"), 0o644); err != nil {
		t.Fatal(err)
	}

	profile := domain.StrategyProfiles[domain.StrategyBalanced]
	jobs := BuildJobs([]domain.VideoMeta{
		{Path: original, Height: 1080, Duration: time.Second},
	}, domain.StrategyBalanced, profile, domain.Res1080p, "_compressed", true)

	if len(jobs) != 1 {
		t.Fatalf("expected one job after stale tmp cleanup, got %d", len(jobs))
	}
	if _, err := os.Stat(tmpOutput); !os.IsNotExist(err) {
		t.Fatalf("expected stale tmp output to be removed, got err=%v", err)
	}
}

func TestBuildJobsDeduplicatesCollidingOutputPaths(t *testing.T) {
	root := t.TempDir()
	clipMov := filepath.Join(root, "clip.mov")
	clipMp4 := filepath.Join(root, "clip.mp4")

	if err := os.WriteFile(clipMov, []byte("source-mov"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(clipMp4, []byte("source-mp4"), 0o644); err != nil {
		t.Fatal(err)
	}

	profile := domain.StrategyProfiles[domain.StrategyBalanced]
	jobs := BuildJobs([]domain.VideoMeta{
		{Path: clipMov, Height: 1080, Duration: time.Second},
		{Path: clipMp4, Height: 1080, Duration: time.Second},
	}, domain.StrategyBalanced, profile, domain.Res1080p, "_compressed", true)

	if len(jobs) != 1 {
		t.Fatalf("expected 1 job (collision deduped), got %d", len(jobs))
	}
}

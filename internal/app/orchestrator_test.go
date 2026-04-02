package app

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/theSearchLife/videoCompressor/internal/domain"
)

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
	}, profile, domain.Res1080p, "_compressed", true)

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
	}, profile, domain.Res720p, "_compressed", true)

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
	}, profile, domain.Res1080p, "_compressed", true)

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
	}, profile, domain.Res1080p, "_compressed", true)

	if len(jobs) != 1 {
		t.Fatalf("expected 1 job (collision deduped), got %d", len(jobs))
	}
}

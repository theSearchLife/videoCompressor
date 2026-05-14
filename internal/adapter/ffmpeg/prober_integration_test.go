package ffmpeg

import (
	"context"
	"os"
	"testing"
)

func TestProberDetectsSamplesFromEnv(t *testing.T) {
	slogPath := os.Getenv("VC_SAMPLE_SLOG3")
	normalPath := os.Getenv("VC_SAMPLE_NORMAL")
	if slogPath == "" || normalPath == "" {
		t.Skip("set VC_SAMPLE_SLOG3 and VC_SAMPLE_NORMAL to run sample metadata detection")
	}

	prober := NewProber()
	slogMeta, err := prober.Probe(context.Background(), slogPath)
	if err != nil {
		t.Fatalf("probe S-Log3 sample: %v", err)
	}
	if !slogMeta.SLog3 {
		t.Fatalf("expected S-Log3 sample to be detected, got %+v", slogMeta)
	}

	normalMeta, err := prober.Probe(context.Background(), normalPath)
	if err != nil {
		t.Fatalf("probe normal sample: %v", err)
	}
	if normalMeta.SLog3 {
		t.Fatalf("expected normal sample not to be detected as S-Log3, got %+v", normalMeta)
	}
}

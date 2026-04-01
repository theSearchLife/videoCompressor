package domain

import "testing"

func TestEffectiveResolutionDoesNotUpscale(t *testing.T) {
	if got := EffectiveResolution(720, Res1080p); got != Res720p {
		t.Fatalf("expected 720p, got %q", got)
	}

	if got := EffectiveResolution(2160, Res1080p); got != Res1080p {
		t.Fatalf("expected 1080p, got %q", got)
	}
}

func TestAssessOutputFilenameUsesClientFriendlyCodecSlug(t *testing.T) {
	profile := Profile{Codec: "libx265", CRF: 26, Preset: "slow"}
	got := AssessOutputFilename("clip.mov", profile, Res1080p)
	want := "clip_h265_crf26_slow_1080p.mp4"
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func TestScaleFilterOmitsNoOpScale(t *testing.T) {
	if got := ScaleFilter(720, Res1080p); got != "" {
		t.Fatalf("expected empty scale filter, got %q", got)
	}
}

func TestTempOutputPathAppendsTmpSuffixToFinalPath(t *testing.T) {
	got := TempOutputPath("/videos/clip_1080p.mp4")
	want := "/videos/clip_1080p.mp4.tmp"
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

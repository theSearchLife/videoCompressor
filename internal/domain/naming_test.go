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

func TestEffectiveResolutionKeepsOriginalWhenTargetEmpty(t *testing.T) {
	if got := EffectiveResolution(1080, ""); got != "" {
		t.Fatalf("expected empty (keep original), got %q", got)
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

func TestScaleFilterReturnsEmptyForKeepOriginal(t *testing.T) {
	if got := ScaleFilter(1080, ""); got != "" {
		t.Fatalf("expected empty scale filter for keep-original, got %q", got)
	}
}

func TestCompressOutputPathUsesSuffix(t *testing.T) {
	got := CompressOutputPath("/videos/clip.mov", "_compressed")
	want := "/videos/clip_compressed.mp4"
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func TestSplitOutputPathDetectsSuffix(t *testing.T) {
	base, ok := SplitOutputPath("/videos/clip_compressed.mp4", "_compressed")
	if !ok {
		t.Fatal("expected output path to be detected")
	}
	if base != "clip" {
		t.Fatalf("expected base %q, got %q", "clip", base)
	}
}

func TestSplitOutputPathRejectsNonMatchingSuffix(t *testing.T) {
	_, ok := SplitOutputPath("/videos/clip_other.mp4", "_compressed")
	if ok {
		t.Fatal("expected non-matching suffix to be rejected")
	}
}

func TestTempOutputPathAppendsTmpSuffixToFinalPath(t *testing.T) {
	got := TempOutputPath("/videos/clip_compressed.mp4")
	want := "/videos/clip_compressed.mp4.tmp"
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func TestBaseOutputPathUsesUnsuffixedMP4Name(t *testing.T) {
	got := BaseOutputPath("/videos/clip.mov")
	want := "/videos/clip.mp4"
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

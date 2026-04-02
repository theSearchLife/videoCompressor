package main

import (
	"bytes"
	"strings"
	"testing"

	"github.com/theSearchLife/videoCompressor/internal/domain"
)

func TestResolveCompressSettingsUsesInteractiveChoices(t *testing.T) {
	// Choose: strategy=1(quality), resolution=3(1080p), fps=2(24), audio=3(medium), suffix=default, skip=1(yes)
	in := strings.NewReader("1\n3\n2\n3\n\n1\n")
	var out bytes.Buffer

	settings, err := resolveCompressSettings(compressFlags{}, in, &out, true)
	if err != nil {
		t.Fatal(err)
	}
	if settings.strategy != domain.StrategyQuality {
		t.Fatalf("expected strategy quality, got %q", settings.strategy)
	}
	if settings.resolution != domain.Res1080p {
		t.Fatalf("expected resolution 1080p, got %q", settings.resolution)
	}
	if settings.fps != 24 {
		t.Fatalf("expected fps 24, got %d", settings.fps)
	}
	if settings.audio != domain.AudioMedium {
		t.Fatalf("expected audio medium, got %q", settings.audio)
	}
	if settings.suffix != "_compressed" {
		t.Fatalf("expected suffix _compressed, got %q", settings.suffix)
	}
	if !settings.skipConverted {
		t.Fatal("expected skipConverted to be true")
	}
	if !strings.Contains(out.String(), "Compression strategy") {
		t.Fatalf("expected prompt output, got %q", out.String())
	}
}

func TestResolveCompressSettingsFallsBackToDefaultsWhenNonInteractive(t *testing.T) {
	settings, err := resolveCompressSettings(compressFlags{}, strings.NewReader(""), &bytes.Buffer{}, false)
	if err != nil {
		t.Fatal(err)
	}
	if settings.strategy != domain.StrategyBalanced {
		t.Fatalf("expected default strategy balanced, got %q", settings.strategy)
	}
	if settings.resolution != "" {
		t.Fatalf("expected default resolution keep-original, got %q", settings.resolution)
	}
	if settings.fps != 0 {
		t.Fatalf("expected default fps 0, got %d", settings.fps)
	}
	if settings.audio != domain.AudioKeep {
		t.Fatalf("expected default audio keep, got %q", settings.audio)
	}
	if settings.suffix != "_compressed" {
		t.Fatalf("expected default suffix _compressed, got %q", settings.suffix)
	}
	if !settings.skipConverted {
		t.Fatal("expected default skipConverted to be true")
	}
}

func TestResolveCompressSettingsLeavesProvidedFlagsUntouched(t *testing.T) {
	settings, err := resolveCompressSettings(compressFlags{
		strategy:      "quality",
		resolution:    "4k",
		fps:           "30",
		audio:         "high",
		suffix:        "_small",
		skipConverted: "no",
	}, strings.NewReader(""), &bytes.Buffer{}, true)
	if err != nil {
		t.Fatal(err)
	}
	if settings.strategy != domain.StrategyQuality {
		t.Fatalf("expected quality, got %q", settings.strategy)
	}
	if settings.resolution != domain.Res4K {
		t.Fatalf("expected 4k, got %q", settings.resolution)
	}
	if settings.fps != 30 {
		t.Fatalf("expected fps 30, got %d", settings.fps)
	}
	if settings.audio != domain.AudioHigh {
		t.Fatalf("expected high, got %q", settings.audio)
	}
	if settings.suffix != "_small" {
		t.Fatalf("expected _small, got %q", settings.suffix)
	}
	if settings.skipConverted {
		t.Fatal("expected skipConverted to be false")
	}
}

func TestResolveCompressSettingsFpsZeroDoesNotPrompt(t *testing.T) {
	// --fps 0 should be treated as "keep original", not trigger a prompt
	settings, err := resolveCompressSettings(compressFlags{
		fps: "0",
	}, strings.NewReader(""), &bytes.Buffer{}, true)
	if err != nil {
		t.Fatal(err)
	}
	if settings.fps != 0 {
		t.Fatalf("expected fps 0, got %d", settings.fps)
	}
}

func TestResolveCompressSettingsRejectsInvalidSkipConverted(t *testing.T) {
	_, err := resolveCompressSettings(compressFlags{
		skipConverted: "typo",
	}, strings.NewReader(""), &bytes.Buffer{}, false)
	if err == nil {
		t.Fatal("expected error for invalid --skip-converted value")
	}
}

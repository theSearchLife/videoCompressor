package domain

import (
	"testing"
	"time"
)

func TestSelectCRFDynamic(t *testing.T) {
	tests := []struct {
		name     string
		strategy CompressionStrategy
		source   VideoMeta
		wantCRF  int
	}{
		// Regression: 20210926_100615_1.mp4 — Samsung phone H.264, 14.1 Mbps,
		// 1080p, ~30fps, 110s. Previously produced 2x larger output with fixed CRF 26.
		// Exact ffprobe values: size=194171921, duration=109.816s, fps=411875/13727≈29.997.
		{
			name:     "samsung phone 20210926_100615_1 balanced",
			strategy: StrategyBalanced,
			source: VideoMeta{
				Width: 1920, Height: 1080,
				Size: 194171921, Duration: 109816 * time.Millisecond,
				FrameRate: 411875.0 / 13727.0,
			},
			wantCRF: 30, // norm ~6.8 → base(26) + 4
		},
		{
			name:     "samsung phone 20210926_100615_1 quality",
			strategy: StrategyQuality,
			source: VideoMeta{
				Width: 1920, Height: 1080,
				Size: 194171921, Duration: 109816 * time.Millisecond,
				FrameRate: 411875.0 / 13727.0,
			},
			wantCRF: 26, // norm ~6.8 → base(22) + 4
		},
		{
			name:     "samsung phone 20210926_100615_1 size",
			strategy: StrategySizePriority,
			source: VideoMeta{
				Width: 1920, Height: 1080,
				Size: 194171921, Duration: 109816 * time.Millisecond,
				FrameRate: 411875.0 / 13727.0,
			},
			wantCRF: 34, // norm ~6.8 → base(30) + 4
		},
		{
			name:     "raw 4K footage at 100Mbps gets base+2",
			strategy: StrategyBalanced,
			source: VideoMeta{
				Width: 3840, Height: 2160,
				Size: 1250000000, Duration: 100 * time.Second,
				FrameRate: 30,
			},
			wantCRF: 28, // norm ~12 → base(26) + 2
		},
		{
			name:     "very high bitrate prosumer gets base CRF",
			strategy: StrategyBalanced,
			source: VideoMeta{
				Width: 1920, Height: 1080,
				Size: 500000000, Duration: 30 * time.Second,
				FrameRate: 30,
			},
			wantCRF: 26, // norm ~64 → base(26)
		},
		{
			name:     "streaming rip at low bitrate gets base+6",
			strategy: StrategyBalanced,
			source: VideoMeta{
				Width: 1920, Height: 1080,
				Size: 50000000, Duration: 120 * time.Second,
				FrameRate: 24,
			},
			wantCRF: 32, // norm ~2.0 → base(26) + 6
		},
		{
			name:     "zero duration returns base CRF",
			strategy: StrategyBalanced,
			source:   VideoMeta{Width: 1920, Height: 1080, Size: 100000},
			wantCRF:  26,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SelectCRF(tt.strategy, tt.source)
			if got != tt.wantCRF {
				t.Errorf("SelectCRF(%s) = %d, want %d", tt.strategy, got, tt.wantCRF)
			}
		})
	}
}

func TestAssessCompression(t *testing.T) {
	samsungPhone := VideoMeta{
		Codec: "h264",
		Width: 1920, Height: 1080,
		Size: 194171921, Duration: 109816 * time.Millisecond,
		FrameRate: 411875.0 / 13727.0,
	}

	t.Run("quality warns for already compressed phone source", func(t *testing.T) {
		profile := ApplyAudioMode(StrategyProfiles[StrategyQuality], AudioKeep)
		profile.FrameRate = 0
		got := AssessCompression(StrategyQuality, samsungPhone, profile, "")
		if !got.Skip || got.Message == "" {
			t.Fatalf("expected skip advice for quality mode, got %+v", got)
		}
	})

	t.Run("balanced does not warn for moderate headroom h264", func(t *testing.T) {
		profile := ApplyAudioMode(StrategyProfiles[StrategyBalanced], AudioKeep)
		profile.FrameRate = 0
		got := AssessCompression(StrategyBalanced, samsungPhone, profile, "")
		if got != (CompressionAdvice{}) {
			t.Fatalf("expected no advice, got %+v", got)
		}
	})

	t.Run("size does not warn", func(t *testing.T) {
		profile := ApplyAudioMode(StrategyProfiles[StrategySizePriority], AudioKeep)
		profile.FrameRate = 0
		got := AssessCompression(StrategySizePriority, samsungPhone, profile, "")
		if got != (CompressionAdvice{}) {
			t.Fatalf("expected no advice, got %+v", got)
		}
	})

	t.Run("downscale suppresses skip", func(t *testing.T) {
		profile := ApplyAudioMode(StrategyProfiles[StrategyQuality], AudioKeep)
		profile.FrameRate = 0
		got := AssessCompression(StrategyQuality, samsungPhone, profile, Res720p)
		if got != (CompressionAdvice{}) {
			t.Fatalf("expected no advice, got %+v", got)
		}
	})

	t.Run("hevc source skips at original settings", func(t *testing.T) {
		hevcSource := VideoMeta{
			Codec: "hevc",
			Width: 1920, Height: 1080,
			Size: 50000000, Duration: 120 * time.Second,
			FrameRate: 30,
		}
		profile := ApplyAudioMode(StrategyProfiles[StrategyBalanced], AudioKeep)
		profile.FrameRate = 0
		got := AssessCompression(StrategyBalanced, hevcSource, profile, "")
		if !got.Skip || got.Message == "" {
			t.Fatalf("expected skip advice for low-headroom hevc source, got %+v", got)
		}
	})
}

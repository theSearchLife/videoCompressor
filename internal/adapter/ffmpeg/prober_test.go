package ffmpeg

import "testing"

func TestDetectSLog3FromMediaInfoTransferCharacteristics(t *testing.T) {
	out := []byte(`{
		"media": {
			"@ref": "/videos/clip.mp4",
			"track": [{
				"@type": "Other",
				"extra": {
					"TransferCharacteristics_FirstFrame": "0E06040101010605",
					"ColorPrimaries_FirstFrame": "BT.709"
				}
			}]
		}
	}`)

	if !detectSLog3FromJSON(out) {
		t.Fatal("expected S-Log3.Cine transfer characteristics to be detected")
	}
}

func TestDetectSLog3FromFullSMPTELabel(t *testing.T) {
	out := []byte(`{
		"streams": [{
			"tags": {
				"CaptureGammaEquation": "06.0E.2B.34.04.01.01.06.0E.06.04.01.01.01.06.04"
			}
		}]
	}`)

	if !detectSLog3FromJSON(out) {
		t.Fatal("expected full Sony S-Log3 SMPTE label to be detected")
	}
}

func TestDetectSLog3IgnoresFilePath(t *testing.T) {
	out := []byte(`{
		"format": {
			"filename": "/tmp/slog3/not-log.mp4",
			"tags": {"major_brand": "mp42"}
		},
		"streams": [{
			"codec_type": "video",
			"color_transfer": "bt709"
		}]
	}`)

	if detectSLog3FromJSON(out) {
		t.Fatal("expected S-Log3-looking path to be ignored")
	}
}

func TestDetectSLog3DoesNotMatchSLog2(t *testing.T) {
	out := []byte(`{
		"media": {
			"track": [{
				"@type": "Other",
				"extra": {"TransferCharacteristics_FirstFrame": "0E06040101010508"}
			}]
		}
	}`)

	if detectSLog3FromJSON(out) {
		t.Fatal("expected Sony S-Log2 transfer characteristics not to match S-Log3")
	}
}

func TestBitDepthFromPixelFormat(t *testing.T) {
	if got := bitDepth("", "yuv422p10le"); got != 10 {
		t.Fatalf("expected 10-bit yuv422p10le, got %d", got)
	}
	if got := bitDepth("8", "yuvj420p"); got != 8 {
		t.Fatalf("expected bits_per_raw_sample to win, got %d", got)
	}
}

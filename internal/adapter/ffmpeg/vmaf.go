package ffmpeg

import (
	"context"
	"fmt"
	"os/exec"
	"regexp"
	"strconv"

	"github.com/theSearchLife/videoCompressor/internal/domain"
)

type VMAFScorer struct{}

func NewVMAFScorer() *VMAFScorer {
	return &VMAFScorer{}
}

var vmafScoreRe = regexp.MustCompile(`VMAF score:\s*([\d.]+)`)

// Score calculates the VMAF score between the encoded output and the original source.
// Both inputs must be at the same resolution, so we scale the reference to match the output.
func (v *VMAFScorer) Score(ctx context.Context, source string, encoded string, outputRes domain.Resolution) (float64, error) {
	targetH := outputRes.Height()

	// libvmaf filter: first input is distorted (encoded), second is reference (source).
	// We scale the reference to match the encoded resolution since VMAF requires equal dimensions.
	filterComplex := fmt.Sprintf(
		"[0:v]setpts=PTS-STARTPTS[dist];"+
			"[1:v]scale=-2:%d,setpts=PTS-STARTPTS[ref];"+
			"[dist][ref]libvmaf=n_threads=4",
		targetH,
	)

	cmd := exec.CommandContext(ctx, "ffmpeg",
		"-i", encoded,
		"-i", source,
		"-filter_complex", filterComplex,
		"-f", "null", "-",
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return 0, fmt.Errorf("vmaf score: %w\n%s", err, string(output))
	}

	matches := vmafScoreRe.FindSubmatch(output)
	if len(matches) < 2 {
		return 0, fmt.Errorf("could not parse VMAF score from output")
	}

	score, err := strconv.ParseFloat(string(matches[1]), 64)
	if err != nil {
		return 0, fmt.Errorf("parse vmaf score: %w", err)
	}

	return score, nil
}

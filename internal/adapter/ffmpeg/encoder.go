package ffmpeg

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/theSearchLife/videoCompressor/internal/domain"
)

type Encoder struct{}

func NewEncoder() *Encoder {
	return &Encoder{}
}

func (e *Encoder) Encode(ctx context.Context, job domain.Job, onProgress func(float64)) error {
	args := buildArgs(job)

	cmd := exec.CommandContext(ctx, "ffmpeg", args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("stdout pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start ffmpeg: %w", err)
	}

	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		line := scanner.Text()
		if p, ok := parseProgress(line, job.Input.Duration); ok && onProgress != nil {
			onProgress(p)
		}
	}
	if err := scanner.Err(); err != nil {
		os.Remove(job.OutputPath)
		return fmt.Errorf("read ffmpeg progress: %w", err)
	}

	if err := cmd.Wait(); err != nil {
		// Clean up partial output on failure
		os.Remove(job.OutputPath)
		message := strings.TrimSpace(stderr.String())
		if message == "" {
			return fmt.Errorf("ffmpeg encode: %w", err)
		}
		return fmt.Errorf("ffmpeg encode: %w: %s", err, message)
	}

	return nil
}

func buildArgs(job domain.Job) []string {
	args := []string{
		"-y",
		"-i", job.Input.Path,
		"-c:v", job.Profile.Codec,
		"-crf", strconv.Itoa(job.Profile.CRF),
		"-preset", job.Profile.Preset,
	}

	if filter := domain.ScaleFilter(job.Input.Height, job.Resolution); filter != "" {
		args = append(args, "-vf", filter)
	}

	if job.Profile.AudioCodec == "copy" {
		args = append(args, "-c:a", "copy")
	} else {
		args = append(args, "-c:a", job.Profile.AudioCodec)
		if job.Profile.AudioBitrate != "" {
			args = append(args, "-b:a", job.Profile.AudioBitrate)
		}
	}

	args = append(args,
		"-movflags", "+faststart",
		"-progress", "pipe:1",
		job.OutputPath,
	)

	return args
}

func parseProgress(line string, totalDuration time.Duration) (float64, bool) {
	if !strings.HasPrefix(line, "out_time_us=") {
		return 0, false
	}
	us, err := strconv.ParseInt(strings.TrimPrefix(line, "out_time_us="), 10, 64)
	if err != nil || totalDuration == 0 {
		return 0, false
	}
	current := time.Duration(us) * time.Microsecond
	progress := float64(current) / float64(totalDuration)
	if progress > 1.0 {
		progress = 1.0
	}
	return progress, true
}

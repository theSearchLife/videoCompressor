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

func (e *Encoder) Encode(ctx context.Context, job domain.Job, onProgress func(float64)) (err error) {
	args := buildArgs(job)
	tempOutputPath := domain.TempOutputPath(job.OutputPath)

	defer func() {
		if err != nil {
			if rmErr := os.Remove(tempOutputPath); rmErr != nil && !os.IsNotExist(rmErr) {
				err = fmt.Errorf("%w (also failed to remove temp output %s: %v)", err, tempOutputPath, rmErr)
			}
		}
	}()

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
		return fmt.Errorf("read ffmpeg progress: %w", err)
	}

	if err := cmd.Wait(); err != nil {
		message := summariseFFmpegError(stderr.String())
		if message == "" {
			return fmt.Errorf("ffmpeg encode: %w", err)
		}
		return fmt.Errorf("ffmpeg encode: %s", message)
	}

	if err := os.Rename(tempOutputPath, job.OutputPath); err != nil {
		return fmt.Errorf("rename completed output: %w", err)
	}

	return nil
}

// summariseFFmpegError extracts the most actionable line from ffmpeg's stderr.
// ffmpeg emits hundreds of progress and informational lines; we want the actual
// error so it can be surfaced in the summary instead of a wall of noise.
func summariseFFmpegError(stderr string) string {
	stderr = strings.TrimSpace(stderr)
	if stderr == "" {
		return ""
	}
	lines := strings.Split(stderr, "\n")
	for i := len(lines) - 1; i >= 0; i-- {
		line := strings.TrimSpace(lines[i])
		lower := strings.ToLower(line)
		if line == "" {
			continue
		}
		if strings.Contains(lower, "error") || strings.Contains(lower, "invalid") || strings.Contains(lower, "could not") || strings.Contains(lower, "no such") || strings.Contains(lower, "failed") {
			return line
		}
	}
	last := strings.TrimSpace(lines[len(lines)-1])
	if last != "" {
		return last
	}
	return stderr
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

	if job.Profile.FrameRate > 0 {
		args = append(args, "-r", strconv.Itoa(job.Profile.FrameRate))
	}

	if job.Profile.Codec == "libx265" {
		args = append(args, "-tag:v", "hvc1")
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
		"-f", "mp4",
		"-progress", "pipe:1",
		domain.TempOutputPath(job.OutputPath),
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

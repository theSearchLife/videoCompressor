package adapter

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/theSearchLife/videoCompressor/internal/domain"
)

// fallbackProgressStepPct is the percent step at which non-TTY runs emit a
// plain progress log line (TTY runs update a single line in place instead).
const fallbackProgressStepPct = 10.0

type jobState struct {
	startedAt    time.Time
	lastStepPct  float64
	lastReported time.Time
}

type LogReporter struct {
	live   *liveProgress
	logger *log.Logger
	mu     sync.Mutex
	jobs   map[int]*jobState
}

// NewLogReporter wires output to stdout and reroutes Go's default log package
// through the live printer so log.Printf lines always land above the live
// region.
func NewLogReporter() *LogReporter {
	r := newLogReporter(os.Stdout)
	log.SetOutput(r.live)
	return r
}

// newLogReporter is the testable constructor: tests pass a buffer in place of
// stdout and skip the global log.SetOutput side effect.
func newLogReporter(out io.Writer) *LogReporter {
	live := newLiveProgress(out)
	logger := log.New(live, "", log.Ltime)
	return &LogReporter{live: live, logger: logger, jobs: make(map[int]*jobState)}
}

func (r *LogReporter) JobStarted(job domain.Job) {
	r.mu.Lock()
	r.jobs[job.ID] = &jobState{startedAt: time.Now()}
	r.mu.Unlock()
	r.logger.Printf("[%d] START: %s -> %s", job.ID, filepath.Base(job.Input.Path), filepath.Base(job.OutputPath))
	r.live.addLine(job.ID, formatLiveLine(job, 0, 0, 0, r.live.lineWidth()))
}

func (r *LogReporter) JobProgress(job domain.Job, progress float64) {
	if progress < 0 {
		progress = 0
	}
	now := time.Now()

	r.mu.Lock()
	state, ok := r.jobs[job.ID]
	if !ok {
		state = &jobState{startedAt: now}
		r.jobs[job.ID] = state
	}
	elapsed := now.Sub(state.startedAt)
	r.mu.Unlock()

	eta := estimateETA(elapsed, progress)

	if r.live.tty {
		r.live.updateLine(job.ID, formatLiveLine(job, progress, elapsed, eta, r.live.lineWidth()))
		return
	}

	// Non-TTY fallback: log a fresh line each time we cross a step boundary
	// or every 15s, so captured logs still show progression.
	pct := progress * 100
	r.mu.Lock()
	defer r.mu.Unlock()
	if pct < state.lastStepPct+fallbackProgressStepPct && now.Sub(state.lastReported) < 15*time.Second {
		return
	}
	state.lastStepPct = pct
	state.lastReported = now
	r.logger.Printf("[%d] %s: %3.0f%%  elapsed %s  eta %s",
		job.ID, filepath.Base(job.OutputPath), pct,
		formatDuration(elapsed), formatDuration(eta))
}

func (r *LogReporter) JobFinished(job domain.Job, result domain.Result) {
	r.live.removeLine(job.ID)
	r.mu.Lock()
	delete(r.jobs, job.ID)
	r.mu.Unlock()

	name := filepath.Base(job.Input.Path)
	sizeSummary := fmt.Sprintf("%s -> %s", formatSize(result.InputSize), formatSize(result.OutputSize))
	encodeTime := formatDuration(result.EncodeTime)
	if result.Error != nil {
		r.logger.Printf("[%d] FAIL: %s  %s  (%s): %v", job.ID, name, sizeSummary, encodeTime, result.Error)
		return
	}
	reduction := result.Reduction() * 100
	if reduction < 20 {
		r.logger.Printf("[%d] WARN: %s  %s -> %s (%.1f%% reduction, %s — minimal savings, consider size profile)",
			job.ID, name,
			formatSize(result.InputSize), formatSize(result.OutputSize), reduction, encodeTime)
	} else {
		r.logger.Printf("[%d] DONE: %s  %s -> %s (%.1f%% reduction, %s)",
			job.ID, name,
			formatSize(result.InputSize), formatSize(result.OutputSize), reduction, encodeTime)
	}
}

func (r *LogReporter) Summary(results []domain.Result, skipped int) {
	var done, failed int
	var totalInput, totalOutput int64
	var failures []domain.Result
	for _, res := range results {
		if res.Error != nil {
			failed++
			failures = append(failures, res)
		} else {
			done++
			totalInput += res.InputSize
			totalOutput += res.OutputSize
		}
	}
	total := len(results) + skipped
	skippedPart := ""
	if skipped > 0 {
		skippedPart = fmt.Sprintf(", %d skipped", skipped)
	}
	if totalInput > 0 {
		reduction := (1 - float64(totalOutput)/float64(totalInput)) * 100
		r.logger.Printf("Summary: %d done, %d failed%s, %d total | %s -> %s (%.1f%% reduction)",
			done, failed, skippedPart, total,
			formatSize(totalInput), formatSize(totalOutput), reduction)
	} else {
		r.logger.Printf("Summary: %d done, %d failed%s, %d total", done, failed, skippedPart, total)
	}

	if len(failures) == 0 {
		return
	}
	r.logger.Printf("Failed files (%d):", len(failures))
	for _, res := range failures {
		r.logger.Printf("  - %s: %s", filepath.Base(res.Job.Input.Path), errorReason(res.Error))
	}
}

func formatLiveLine(job domain.Job, progress float64, elapsed, eta time.Duration, maxWidth int) string {
	pct := progress * 100
	bar := renderBar(progress, 20)
	prefix := fmt.Sprintf("[%d] %s %3.0f%%  ", job.ID, bar, pct)
	suffix := fmt.Sprintf("  elapsed %s  eta %s", formatDuration(elapsed), formatDuration(eta))
	name := filepath.Base(job.OutputPath)

	if maxWidth > 0 {
		nameWidth := maxWidth - textWidth(prefix) - textWidth(suffix)
		name = compactMiddle(name, nameWidth)
	}

	return prefix + name + suffix
}

func renderBar(progress float64, width int) string {
	if progress < 0 {
		progress = 0
	}
	if progress > 1 {
		progress = 1
	}
	filled := int(progress * float64(width))
	if filled > width {
		filled = width
	}
	bar := make([]byte, 0, width+2)
	bar = append(bar, '[')
	for i := 0; i < width; i++ {
		switch {
		case i < filled:
			bar = append(bar, '#')
		case i == filled:
			bar = append(bar, '>')
		default:
			bar = append(bar, '-')
		}
	}
	bar = append(bar, ']')
	return string(bar)
}

func errorReason(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

func estimateETA(elapsed time.Duration, progress float64) time.Duration {
	if progress <= 0.001 {
		return 0
	}
	if progress >= 1.0 {
		return 0
	}
	total := time.Duration(float64(elapsed) / progress)
	return total - elapsed
}

func formatDuration(d time.Duration) string {
	if d <= 0 {
		return "--:--"
	}
	totalSec := int(d.Round(time.Second).Seconds())
	h := totalSec / 3600
	m := (totalSec % 3600) / 60
	s := totalSec % 60
	if h > 0 {
		return fmt.Sprintf("%d:%02d:%02d", h, m, s)
	}
	return fmt.Sprintf("%02d:%02d", m, s)
}

func formatSize(bytes int64) string {
	if bytes == 0 {
		return "0 B"
	}
	units := []string{"B", "KB", "MB", "GB"}
	i := 0
	size := float64(bytes)
	for size >= 1024 && i < len(units)-1 {
		size /= 1024
		i++
	}
	return fmt.Sprintf("%.1f %s", size, units[i])
}

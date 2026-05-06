package adapter

import (
	"fmt"
	"log"
	"path/filepath"
	"sync"
	"time"

	"github.com/theSearchLife/videoCompressor/internal/domain"
)

// progressStepPct controls how often a progress line is printed per job.
// With workers > 1 we cannot use \r overwrite (lines collide and Windows
// PowerShell does not always honour carriage returns), so we emit a fresh
// log line each step and tag it with the job ID + filename.
const progressStepPct = 10.0

type jobState struct {
	startedAt    time.Time
	lastStepPct  float64
	lastReported time.Time
}

type LogReporter struct {
	mu    sync.Mutex
	jobs  map[int]*jobState
	total int
}

func NewLogReporter() *LogReporter {
	return &LogReporter{jobs: make(map[int]*jobState)}
}

// SetTotal lets callers tell the reporter how many jobs are in this batch so
// progress lines can show "[i/N]" — purely cosmetic.
func (r *LogReporter) SetTotal(n int) {
	r.mu.Lock()
	r.total = n
	r.mu.Unlock()
}

func (r *LogReporter) JobStarted(job domain.Job) {
	r.mu.Lock()
	r.jobs[job.ID] = &jobState{startedAt: time.Now()}
	r.mu.Unlock()
	log.Printf("[%d] START: %s -> %s", job.ID, filepath.Base(job.Input.Path), filepath.Base(job.OutputPath))
}

func (r *LogReporter) JobProgress(job domain.Job, progress float64) {
	if progress < 0 {
		progress = 0
	}
	pct := progress * 100
	now := time.Now()

	r.mu.Lock()
	state, ok := r.jobs[job.ID]
	if !ok {
		state = &jobState{startedAt: now}
		r.jobs[job.ID] = state
	}
	// throttle: report when we cross the next progressStepPct boundary,
	// or at least every 15s while encoding is still active.
	if pct < state.lastStepPct+progressStepPct && now.Sub(state.lastReported) < 15*time.Second {
		r.mu.Unlock()
		return
	}
	state.lastStepPct = pct
	state.lastReported = now
	elapsed := now.Sub(state.startedAt)
	r.mu.Unlock()

	eta := estimateETA(elapsed, progress)
	log.Printf("[%d] %s: %3.0f%%  elapsed %s  eta %s",
		job.ID, filepath.Base(job.OutputPath), pct,
		formatDuration(elapsed), formatDuration(eta))
}

func (r *LogReporter) JobFinished(job domain.Job, result domain.Result) {
	r.mu.Lock()
	delete(r.jobs, job.ID)
	r.mu.Unlock()

	name := filepath.Base(job.Input.Path)
	sizeSummary := fmt.Sprintf("%s -> %s", formatSize(result.InputSize), formatSize(result.OutputSize))
	encodeTime := formatDuration(result.EncodeTime)
	if result.Error != nil {
		log.Printf("[%d] FAIL: %s  %s  (%s): %v", job.ID, name, sizeSummary, encodeTime, result.Error)
		return
	}
	reduction := result.Reduction() * 100
	if reduction < 20 {
		log.Printf("[%d] WARN: %s  %s -> %s (%.1f%% reduction, %s — minimal savings, consider size profile)",
			job.ID, name,
			formatSize(result.InputSize), formatSize(result.OutputSize), reduction, encodeTime)
	} else {
		log.Printf("[%d] DONE: %s  %s -> %s (%.1f%% reduction, %s)",
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
		log.Printf("Summary: %d done, %d failed%s, %d total | %s -> %s (%.1f%% reduction)",
			done, failed, skippedPart, total,
			formatSize(totalInput), formatSize(totalOutput), reduction)
	} else {
		log.Printf("Summary: %d done, %d failed%s, %d total", done, failed, skippedPart, total)
	}

	if len(failures) == 0 {
		return
	}
	log.Printf("Failed files (%d):", len(failures))
	for _, res := range failures {
		log.Printf("  - %s: %s", filepath.Base(res.Job.Input.Path), errorReason(res.Error))
	}
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

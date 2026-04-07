package adapter

import (
	"fmt"
	"log"
	"path/filepath"

	"github.com/theSearchLife/videoCompressor/internal/domain"
)

type LogReporter struct{}

func NewLogReporter() *LogReporter {
	return &LogReporter{}
}

func (r *LogReporter) JobStarted(job domain.Job) {
	log.Printf("[%d] START: %s → %s", job.ID, filepath.Base(job.Input.Path), filepath.Base(job.OutputPath))
}

func (r *LogReporter) JobProgress(job domain.Job, progress float64) {
	fmt.Printf("\r[%d] %s: %.0f%%", job.ID, filepath.Base(job.OutputPath), progress*100)
}

func (r *LogReporter) JobFinished(job domain.Job, result domain.Result) {
	name := filepath.Base(job.Input.Path)
	sizeSummary := fmt.Sprintf("%s → %s", formatSize(result.InputSize), formatSize(result.OutputSize))
	if result.Error != nil {
		log.Printf("[%d] FAIL: %s  %s: %v", job.ID, name, sizeSummary, result.Error)
		return
	}
	reduction := result.Reduction() * 100
	if reduction < 20 {
		log.Printf("[%d] WARN: %s  %s → %s (%.1f%% reduction — minimal savings, consider size profile)",
			job.ID, name,
			formatSize(result.InputSize), formatSize(result.OutputSize), reduction)
	} else {
		log.Printf("[%d] DONE: %s  %s → %s (%.1f%% reduction)",
			job.ID, name,
			formatSize(result.InputSize), formatSize(result.OutputSize), reduction)
	}
}

func (r *LogReporter) Summary(results []domain.Result) {
	var done, failed int
	var totalInput, totalOutput int64
	for _, res := range results {
		if res.Error != nil {
			failed++
		} else {
			done++
			totalInput += res.InputSize
			totalOutput += res.OutputSize
		}
	}
	if totalInput > 0 {
		reduction := (1 - float64(totalOutput)/float64(totalInput)) * 100
		log.Printf("Summary: %d done, %d failed, %d total | %s → %s (%.1f%% reduction)",
			done, failed, len(results),
			formatSize(totalInput), formatSize(totalOutput), reduction)
	} else {
		log.Printf("Summary: %d done, %d failed, %d total", done, failed, len(results))
	}
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

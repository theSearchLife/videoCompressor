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

func (r *LogReporter) JobFinished(job domain.Job, err error) {
	if err != nil {
		log.Printf("[%d] FAIL: %s: %v", job.ID, filepath.Base(job.OutputPath), err)
	} else {
		log.Printf("[%d] DONE: %s", job.ID, filepath.Base(job.OutputPath))
	}
}

func (r *LogReporter) Summary(results []domain.Result) {
	var done, failed, skipped int
	for _, res := range results {
		switch {
		case res.Error != nil:
			failed++
		default:
			done++
		}
	}
	_ = skipped
	log.Printf("Summary: %d done, %d failed, %d total", done, failed, len(results))
}

package app

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/theSearchLife/videoCompressor/internal/domain"
	"github.com/theSearchLife/videoCompressor/internal/port"
)

type Orchestrator struct {
	encoder  port.Encoder
	reporter port.Reporter
	workers  int
}

func NewOrchestrator(encoder port.Encoder, reporter port.Reporter, workers int) *Orchestrator {
	return &Orchestrator{
		encoder:  encoder,
		reporter: reporter,
		workers:  workers,
	}
}

func (o *Orchestrator) Run(ctx context.Context, jobs []domain.Job) []domain.Result {
	results := make([]domain.Result, len(jobs))
	sem := make(chan struct{}, o.workers)
	var wg sync.WaitGroup

	for i, job := range jobs {
		wg.Add(1)
		sem <- struct{}{}
		go func(idx int, j domain.Job) {
			defer wg.Done()
			defer func() { <-sem }()

			o.reporter.JobStarted(j)
			start := time.Now()

			inputSize := j.Input.Size
			err := o.encoder.Encode(ctx, j, func(p float64) {
				o.reporter.JobProgress(j, p)
			})

			var outputSize int64
			if err == nil {
				if info, statErr := os.Stat(j.OutputPath); statErr == nil {
					outputSize = info.Size()
				}
			}

			results[idx] = domain.Result{
				Job:        j,
				InputSize:  inputSize,
				OutputSize: outputSize,
				EncodeTime: time.Since(start),
				Error:      err,
			}

			o.reporter.JobFinished(j, err)
		}(i, job)
	}

	wg.Wait()
	return results
}

func BuildJobs(files []domain.VideoMeta, profile domain.Profile, targetRes domain.Resolution) []domain.Job {
	var jobs []domain.Job
	for i, meta := range files {
		effectiveRes := domain.EffectiveResolution(meta.Height, targetRes)
		outputPath := domain.CompressOutputPath(meta.Path, effectiveRes)

		if _, err := os.Stat(outputPath); err == nil {
			continue // skip existing outputs (idempotent)
		}

		jobs = append(jobs, domain.Job{
			ID:         i,
			Input:      meta,
			OutputPath: outputPath,
			Profile:    profile,
			Resolution: effectiveRes,
			Status:     domain.StatusPending,
		})
	}

	if len(jobs) == 0 {
		fmt.Println("No videos to encode (all outputs already exist or no files found).")
	}

	return jobs
}

package app

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
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
				if inputSize > 0 && outputSize >= inputSize {
					os.Remove(j.OutputPath)
					increase := (float64(outputSize)/float64(inputSize) - 1) * 100
					err = fmt.Errorf("output larger than input (%.0f%% increase): try the size profile for this file", increase)
				}
			}

			result := domain.Result{
				Job:        j,
				InputSize:  inputSize,
				OutputSize: outputSize,
				EncodeTime: time.Since(start),
				Error:      err,
			}
			results[idx] = result

			o.reporter.JobFinished(j, result)
		}(i, job)
	}

	wg.Wait()
	return results
}

func BuildJobs(files []domain.VideoMeta, strategy domain.CompressionStrategy, profile domain.Profile, targetRes domain.Resolution, suffix string, skipConverted bool) []domain.Job {
	var jobs []domain.Job
	seen := make(map[string]string) // output path -> first source path

	for i, meta := range files {
		if isDerivedOutputPath(meta.Path, suffix) {
			log.Printf("SKIP: %s (appears to be a previous output)", meta.Path)
			continue
		}

		outputPath := domain.CompressOutputPath(meta.Path, suffix)
		cleanupTempOutput(outputPath)

		if firstSource, ok := seen[outputPath]; ok {
			log.Printf("WARN: %s and %s both map to %s, skipping %s",
				firstSource, meta.Path, filepath.Base(outputPath), meta.Path)
			continue
		}
		seen[outputPath] = meta.Path

		if skipConverted {
			if _, err := os.Stat(outputPath); err == nil {
				continue // skip existing outputs (idempotent)
			}
		}

		p := profile
		p.CRF = domain.SelectCRF(strategy, meta)

		jobs = append(jobs, domain.Job{
			ID:         i,
			Input:      meta,
			OutputPath: outputPath,
			Profile:    p,
			Resolution: targetRes,
			Status:     domain.StatusPending,
		})
	}

	if len(jobs) == 0 {
		fmt.Println("No videos to encode (all outputs already exist or no files found).")
	}

	return jobs
}

func cleanupTempOutput(outputPath string) {
	tmpPath := domain.TempOutputPath(outputPath)
	if err := os.Remove(tmpPath); err != nil && !os.IsNotExist(err) {
		return
	}
}

func isDerivedOutputPath(path string, suffix string) bool {
	base, ok := domain.SplitOutputPath(path, suffix)
	if !ok {
		return false
	}

	dir := filepath.Dir(path)
	for _, ext := range domain.RecognizedVideoExtensions() {
		candidate := filepath.Join(dir, base+ext)
		if candidate == path {
			continue
		}
		if _, err := os.Stat(candidate); err == nil && domain.CompressOutputPath(candidate, suffix) == path {
			return true
		}
	}

	return false
}

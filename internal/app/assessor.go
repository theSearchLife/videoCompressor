package app

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/theSearchLife/videoCompressor/internal/domain"
	"github.com/theSearchLife/videoCompressor/internal/port"
	"github.com/theSearchLife/videoCompressor/internal/report"
)

type VMAFScorer interface {
	Score(ctx context.Context, source, encoded string, outputRes domain.Resolution) (float64, error)
}

type Assessor struct {
	scanner  port.Scanner
	prober   port.Prober
	encoder  port.Encoder
	reporter port.Reporter
	vmaf     VMAFScorer
}

func NewAssessor(scanner port.Scanner, prober port.Prober, encoder port.Encoder, reporter port.Reporter, vmaf VMAFScorer) *Assessor {
	return &Assessor{
		scanner:  scanner,
		prober:   prober,
		encoder:  encoder,
		reporter: reporter,
		vmaf:     vmaf,
	}
}

type AssessOptions struct {
	InputDir  string
	OutputDir string
	Matrix    domain.MatrixConfig
	Workers   int
}

func (a *Assessor) Run(ctx context.Context, opts AssessOptions) error {
	timestamp := time.Now().Format("2006-01-02T15-04-05")
	runDir := filepath.Join(opts.OutputDir, timestamp)
	encodedDir := filepath.Join(runDir, "encoded")
	if err := os.MkdirAll(encodedDir, 0o755); err != nil {
		return fmt.Errorf("create output dir: %w", err)
	}

	files, err := a.scanner.Scan(ctx, opts.InputDir, false)
	if err != nil {
		return fmt.Errorf("scan: %w", err)
	}
	if len(files) == 0 {
		return fmt.Errorf("no video files found in %s", opts.InputDir)
	}

	var sources []domain.VideoMeta
	for _, f := range files {
		meta, err := a.prober.Probe(ctx, f)
		if err != nil {
			log.Printf("WARN: skipping %s: %v", f, err)
			continue
		}
		sources = append(sources, meta)
	}

	profiles := opts.Matrix.Profiles()
	total := opts.Matrix.TotalCombinations(len(sources))
	log.Printf("Assessment: %d sources x %d profiles x %d resolutions = %d encodes",
		len(sources), len(profiles), len(opts.Matrix.Resolutions), total)

	jobs := make([]domain.Job, 0, total)
	jobID := 0
	for _, source := range sources {
		for _, profile := range profiles {
			for _, res := range opts.Matrix.Resolutions {
				if ctx.Err() != nil {
					return ctx.Err()
				}

				effectiveRes := domain.EffectiveResolution(source.Height, res)
				inputBase := filepath.Base(source.Path)
				outputName := domain.AssessOutputFilename(inputBase, profile, effectiveRes)
				outputPath := filepath.Join(encodedDir, outputName)

				if _, err := os.Stat(outputPath); err == nil {
					log.Printf("SKIP: %s (already exists)", outputName)
					continue
				}

				jobs = append(jobs, domain.Job{
					ID:         jobID,
					Input:      source,
					OutputPath: outputPath,
					Profile:    profile,
					Resolution: effectiveRes,
					Status:     domain.StatusPending,
				})
				jobID++
			}
		}
	}

	workers := opts.Workers
	if workers < 1 {
		workers = 1
	}

	results := NewOrchestrator(a.encoder, a.reporter, workers).Run(ctx, jobs)
	for i := range results {
		if results[i].Error == nil && a.vmaf != nil {
			outputName := filepath.Base(results[i].Job.OutputPath)
			log.Printf("VMAF: scoring %s...", outputName)
			vmafScore, vmafErr := a.vmaf.Score(ctx, results[i].Job.Input.Path, results[i].Job.OutputPath, results[i].Job.Resolution)
			if vmafErr != nil {
				log.Printf("VMAF WARN: %s: %v", outputName, vmafErr)
			} else {
				results[i].VMAF = vmafScore
			}
		}

		outputName := filepath.Base(results[i].Job.OutputPath)
		if results[i].Error != nil {
			log.Printf("FAIL: %s: %v", outputName, results[i].Error)
			continue
		}

		vmafStr := ""
		if results[i].VMAF > 0 {
			vmafStr = fmt.Sprintf(", VMAF %.1f", results[i].VMAF)
		}
		log.Printf("DONE: %s (%.1f%% reduction, %s%s)",
			outputName, results[i].Reduction()*100, results[i].EncodeTime.Round(time.Second), vmafStr)
	}

	reportPath := filepath.Join(runDir, "report.md")
	csvPath := filepath.Join(runDir, "results.csv")

	if err := report.WriteMarkdown(reportPath, sources, results); err != nil {
		return fmt.Errorf("write report: %w", err)
	}
	if err := report.WriteCSV(csvPath, results); err != nil {
		return fmt.Errorf("write csv: %w", err)
	}

	log.Printf("Assessment complete. Report: %s", reportPath)
	return nil
}

package report

import (
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/theSearchLife/videoCompressor/internal/domain"
)

func WriteMarkdown(path string, sources []domain.VideoMeta, results []domain.Result) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	fmt.Fprintf(f, "# Video Compression Assessment Report\n\n")
	fmt.Fprintf(f, "Generated: %s\n\n", time.Now().Format("2006-01-02 15:04:05"))

	fmt.Fprintf(f, "## Source Files\n\n")
	fmt.Fprintf(f, "| File | Resolution | Codec | Duration | Size |\n")
	fmt.Fprintf(f, "|------|-----------|-------|----------|------|\n")
	for _, s := range sources {
		fmt.Fprintf(f, "| %s | %dx%d | %s | %s | %s |\n",
			filepath.Base(s.Path), s.Width, s.Height, s.Codec,
			s.Duration.Round(time.Second), formatSize(s.Size))
	}

	hasVMAF := false
	for _, r := range results {
		if r.VMAF > 0 {
			hasVMAF = true
			break
		}
	}

	fmt.Fprintf(f, "\n## Results\n\n")

	if hasVMAF {
		fmt.Fprintf(f, "> VMAF scores: 93+ = visually lossless, 85-93 = good, 75-85 = acceptable, <75 = visible loss\n\n")
	}

	bySource := groupBySource(results)
	for _, sourcePath := range sortedSourcePaths(bySource) {
		sourceResults := bySource[sourcePath]
		fmt.Fprintf(f, "### %s\n\n", filepath.Base(sourcePath))

		if hasVMAF {
			fmt.Fprintf(f, "| # | Codec | CRF | Preset | Resolution | Output Size | Reduction | VMAF | Encode Time |\n")
			fmt.Fprintf(f, "|---|-------|-----|--------|-----------|-------------|-----------|------|-------------|\n")
		} else {
			fmt.Fprintf(f, "| # | Codec | CRF | Preset | Resolution | Output Size | Reduction | Encode Time |\n")
			fmt.Fprintf(f, "|---|-------|-----|--------|-----------|-------------|-----------|-------------|\n")
		}

		for i, r := range sourceResults {
			if r.Error != nil {
				if hasVMAF {
					fmt.Fprintf(f, "| %d | %s | %d | %s | %s | FAILED | — | — | — |\n",
						i+1, domain.CodecDisplayName(r.Job.Profile.Codec), r.Job.Profile.CRF,
						r.Job.Profile.Preset, r.Job.Resolution)
				} else {
					fmt.Fprintf(f, "| %d | %s | %d | %s | %s | FAILED | — | — |\n",
						i+1, domain.CodecDisplayName(r.Job.Profile.Codec), r.Job.Profile.CRF,
						r.Job.Profile.Preset, r.Job.Resolution)
				}
				continue
			}

			if hasVMAF {
				vmafStr := "—"
				if r.VMAF > 0 {
					vmafStr = fmt.Sprintf("%.1f", r.VMAF)
				}
				fmt.Fprintf(f, "| %d | %s | %d | %s | %s | %s | %.1f%% | %s | %s |\n",
					i+1,
					domain.CodecDisplayName(r.Job.Profile.Codec),
					r.Job.Profile.CRF,
					r.Job.Profile.Preset,
					r.Job.Resolution,
					formatSize(r.OutputSize),
					r.Reduction()*100,
					vmafStr,
					r.EncodeTime.Round(time.Second),
				)
			} else {
				fmt.Fprintf(f, "| %d | %s | %d | %s | %s | %s | %.1f%% | %s |\n",
					i+1,
					domain.CodecDisplayName(r.Job.Profile.Codec),
					r.Job.Profile.CRF,
					r.Job.Profile.Preset,
					r.Job.Resolution,
					formatSize(r.OutputSize),
					r.Reduction()*100,
					r.EncodeTime.Round(time.Second),
				)
			}
		}
		fmt.Fprintln(f)
	}

	fmt.Fprintf(f, "## Recommendation\n\n")
	fmt.Fprintf(f, "Based on the results above, the recommended profiles are:\n\n")
	fmt.Fprintf(f, "- **High compression:** [to be filled after review]\n")
	fmt.Fprintf(f, "- **Low compression:** [to be filled after review]\n")

	return nil
}

func WriteCSV(path string, results []domain.Result) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	w := csv.NewWriter(f)
	defer w.Flush()

	w.Write([]string{
		"source", "codec", "crf", "preset", "resolution",
		"input_size_bytes", "output_size_bytes", "reduction_pct",
		"encode_time_s", "vmaf", "error",
	})

	for _, r := range results {
		errStr := ""
		if r.Error != nil {
			errStr = r.Error.Error()
		}
		vmafStr := ""
		if r.VMAF > 0 {
			vmafStr = fmt.Sprintf("%.2f", r.VMAF)
		}
		w.Write([]string{
			filepath.Base(r.Job.Input.Path),
			r.Job.Profile.Codec,
			fmt.Sprintf("%d", r.Job.Profile.CRF),
			r.Job.Profile.Preset,
			string(r.Job.Resolution),
			fmt.Sprintf("%d", r.InputSize),
			fmt.Sprintf("%d", r.OutputSize),
			fmt.Sprintf("%.1f", r.Reduction()*100),
			fmt.Sprintf("%.1f", r.EncodeTime.Seconds()),
			vmafStr,
			errStr,
		})
	}

	return nil
}

func groupBySource(results []domain.Result) map[string][]domain.Result {
	m := make(map[string][]domain.Result)
	for _, r := range results {
		m[r.Job.Input.Path] = append(m[r.Job.Input.Path], r)
	}
	return m
}

func sortedSourcePaths(bySource map[string][]domain.Result) []string {
	keys := make([]string, 0, len(bySource))
	for sourcePath := range bySource {
		keys = append(keys, sourcePath)
	}
	sort.Strings(keys)
	return keys
}

func formatSize(bytes int64) string {
	switch {
	case bytes >= 1<<30:
		return fmt.Sprintf("%.1f GB", float64(bytes)/float64(1<<30))
	case bytes >= 1<<20:
		return fmt.Sprintf("%.1f MB", float64(bytes)/float64(1<<20))
	case bytes >= 1<<10:
		return fmt.Sprintf("%.1f KB", float64(bytes)/float64(1<<10))
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}

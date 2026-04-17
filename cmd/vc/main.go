package main

import (
	"bufio"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"regexp"
	"runtime"
	"strconv"
	"strings"

	"github.com/theSearchLife/videoCompressor/internal/adapter"
	"github.com/theSearchLife/videoCompressor/internal/adapter/ffmpeg"
	fsadapter "github.com/theSearchLife/videoCompressor/internal/adapter/fs"
	"github.com/theSearchLife/videoCompressor/internal/app"
	"github.com/theSearchLife/videoCompressor/internal/domain"
	"golang.org/x/term"
)

func main() {
	log.SetFlags(log.Ltime)

	if len(os.Args) < 2 {
		usage()
		os.Exit(1)
	}

	subcommand := os.Args[1]

	switch subcommand {
	case "assess":
		runAssess(os.Args[2:])
	case "cleanup":
		runCleanup(os.Args[2:])
	case "compress":
		runCompress(os.Args[2:])
	case "help", "--help", "-h":
		usage()
	default:
		// Default to compress if first arg looks like a path
		runCompress(os.Args[1:])
	}
}

func runAssess(args []string) {
	fset := flag.NewFlagSet("assess", flag.ExitOnError)
	outputDir := fset.String("output", "./comparison_reports", "Report output directory")
	workers := fset.Int("workers", 1, "Parallel encoding jobs (default 1 for consistent timing)")

	var positional []string
	var flagArgs []string
	for i := 0; i < len(args); i++ {
		if strings.HasPrefix(args[i], "-") {
			flagArgs = append(flagArgs, args[i])
			if i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
				flagArgs = append(flagArgs, args[i+1])
				i++
			}
		} else {
			positional = append(positional, args[i])
		}
	}
	fset.Parse(flagArgs)

	if len(positional) < 1 {
		fmt.Fprintln(os.Stderr, "Usage: vc assess <input-dir> [flags]")
		os.Exit(1)
	}
	inputDir := positional[0]

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	scanner := fsadapter.NewScanner()
	prober := ffmpeg.NewProber()
	encoder := ffmpeg.NewEncoder()
	reporter := adapter.NewLogReporter()
	vmafScorer := ffmpeg.NewVMAFScorer()

	assessor := app.NewAssessor(scanner, prober, encoder, reporter, vmafScorer)

	opts := app.AssessOptions{
		InputDir:  inputDir,
		OutputDir: *outputDir,
		Matrix:    domain.DefaultMatrixConfig(),
		Workers:   *workers,
	}

	if err := assessor.Run(ctx, opts); err != nil {
		log.Fatalf("Assessment failed: %v", err)
	}
}

func runCompress(args []string) {
	flags := flag.NewFlagSet("compress", flag.ExitOnError)
	flagStrategy := flags.String("strategy", "", "Compression strategy: quality, balanced, size")
	flagResolution := flags.String("resolution", "", "Target resolution: original, 720p, 1080p, 4k")
	flagFPS := flags.String("fps", "", "Target frame rate: 0=keep original, 24, 30, 60")
	flagAudio := flags.String("audio", "", "Audio quality: keep, low, medium, high")
	flagSuffix := flags.String("suffix", "", "Output file suffix (default: _compressed)")
	flagSkip := flags.String("skip-converted", "", "Skip already converted files: yes or no (default: yes)")
	workers := flags.Int("workers", runtime.NumCPU()/2, "Parallel encoding jobs")
	dryRun := flags.Bool("dry-run", false, "Show what would be encoded")

	var positional []string
	var flagArgs []string
	for i := 0; i < len(args); i++ {
		if strings.HasPrefix(args[i], "-") {
			flagArgs = append(flagArgs, args[i])
			if args[i] == "--dry-run" || args[i] == "-n" {
				continue
			}
			if i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
				flagArgs = append(flagArgs, args[i+1])
				i++
			}
		} else {
			positional = append(positional, args[i])
		}
	}
	flags.Parse(flagArgs)

	if len(positional) < 1 {
		fmt.Fprintln(os.Stderr, "Usage: vc [compress] <input-dir> [flags]")
		os.Exit(1)
	}
	inputDir := positional[0]

	interactive := isInteractiveInput(os.Stdin)
	settings, err := resolveCompressSettings(
		compressFlags{
			strategy:      *flagStrategy,
			resolution:    *flagResolution,
			fps:           *flagFPS,
			audio:         *flagAudio,
			suffix:        *flagSuffix,
			skipConverted: *flagSkip,
		},
		os.Stdin, os.Stdout, interactive,
	)
	if err != nil {
		log.Fatalf("Resolve settings failed: %v", err)
	}

	profile := domain.StrategyProfiles[settings.strategy]
	profile = domain.ApplyAudioMode(profile, settings.audio)
	profile.FrameRate = settings.fps

	if *workers < 1 {
		*workers = 1
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	scanner := fsadapter.NewScanner()
	prober := ffmpeg.NewProber()
	encoder := ffmpeg.NewEncoder()
	reporter := adapter.NewLogReporter()

	files, err := scanner.Scan(ctx, inputDir)
	if err != nil {
		log.Fatalf("Scan failed: %v", err)
	}
	if len(files) == 0 {
		fmt.Println("No video files found.")
		return
	}

	var metas []domain.VideoMeta
	for _, f := range files {
		meta, err := prober.Probe(ctx, f)
		if err != nil {
			log.Printf("WARN: skipping %s: %v", f, err)
			continue
		}
		metas = append(metas, meta)
	}

	jobs, skipped := app.BuildJobs(metas, settings.strategy, profile, settings.resolution, settings.suffix, settings.skipConverted)
	if len(jobs) == 0 {
		return
	}

	if *dryRun {
		fmt.Printf("Would encode %d files:\n", len(jobs))
		for _, j := range jobs {
			fmt.Printf("  %s -> %s\n", j.Input.Path, j.OutputPath)
		}
		return
	}

	orch := app.NewOrchestrator(encoder, reporter, *workers)
	results := orch.Run(ctx, jobs)
	reporter.Summary(results, skipped)
}

func runCleanup(args []string) {
	flags := flag.NewFlagSet("cleanup", flag.ExitOnError)
	suffix := flags.String("suffix", "", "Suffix used during compression (default: _compressed)")

	var positional []string
	var flagArgs []string
	for i := 0; i < len(args); i++ {
		if strings.HasPrefix(args[i], "-") {
			flagArgs = append(flagArgs, args[i])
			if i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
				flagArgs = append(flagArgs, args[i+1])
				i++
			}
		} else {
			positional = append(positional, args[i])
		}
	}
	flags.Parse(flagArgs)

	if len(positional) < 1 {
		fmt.Fprintln(os.Stderr, "Usage: vc cleanup <input-dir> [flags]")
		os.Exit(1)
	}
	inputDir := positional[0]

	interactive := isInteractiveInput(os.Stdin)
	resolvedSuffix, err := resolveSuffix(*suffix, bufio.NewReader(os.Stdin), os.Stdout, interactive)
	if err != nil {
		log.Fatalf("Resolve suffix failed: %v", err)
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	scanner := fsadapter.NewScanner()
	cleanup := app.NewCleanupService(scanner)

	opts := app.CleanupOptions{
		InputDir: inputDir,
		Suffix:   resolvedSuffix,
	}

	actions, err := cleanup.Plan(ctx, opts)
	if err != nil {
		log.Fatalf("Cleanup failed: %v", err)
	}
	if len(actions) == 0 {
		fmt.Println("No converted outputs matched the cleanup criteria.")
		return
	}

	fmt.Printf("\nFound %d originals with matching converted outputs:\n", len(actions))
	for _, action := range actions {
		fmt.Printf("  %s -> %s\n", action.OriginalPath, action.FinalPath)
	}

	if interactive {
		reader := bufio.NewReader(os.Stdin)
		val, promptErr := promptChoiceValue(reader, os.Stdout, "Delete originals and rename converted files?", []promptChoice{
			{label: "Yes", value: "yes"},
			{label: "No", value: "no"},
		}, "no")
		if promptErr != nil || val != "yes" {
			fmt.Println("Cleanup cancelled.")
			return
		}
	}

	actions, err = cleanup.Run(ctx, opts)
	if err != nil {
		log.Fatalf("Cleanup failed: %v", err)
	}

	fmt.Printf("Cleaned %d converted outputs:\n", len(actions))
	for _, action := range actions {
		fmt.Printf("  %s -> %s\n", action.OriginalPath, action.FinalPath)
	}
}

// --- Settings resolution ---

type compressFlags struct {
	strategy      string
	resolution    string
	fps           string // "" = prompt, "0" = keep original, "24"/"30"/"60"
	audio         string
	suffix        string
	skipConverted string // "yes", "no", or "" (prompt)
}

type compressSettings struct {
	strategy      domain.CompressionStrategy
	resolution    domain.Resolution
	fps           int
	audio         domain.AudioMode
	suffix        string
	skipConverted bool
}

func resolveCompressSettings(flags compressFlags, in io.Reader, out io.Writer, interactive bool) (compressSettings, error) {
	reader := bufio.NewReader(in)
	s := compressSettings{}
	var err error

	// 1. Compression strategy
	if flags.strategy != "" {
		s.strategy, err = parseStrategy(flags.strategy)
		if err != nil {
			return s, err
		}
	} else if interactive {
		val, promptErr := promptChoiceValue(reader, out, "Compression strategy", []promptChoice{
			{label: "Quality (slow)", value: "quality"},
			{label: "Balanced", value: "balanced"},
			{label: "Size (fast)", value: "size"},
		}, "balanced")
		if promptErr != nil {
			return s, promptErr
		}
		s.strategy, _ = parseStrategy(val)
	} else {
		s.strategy = domain.StrategyBalanced
	}

	// 2. Resolution
	if flags.resolution != "" {
		s.resolution, err = parseResolution(flags.resolution)
		if err != nil {
			return s, err
		}
	} else if interactive {
		val, promptErr := promptChoiceValue(reader, out, "Resolution", []promptChoice{
			{label: "Keep original", value: "original"},
			{label: "720p", value: "720p"},
			{label: "1080p", value: "1080p"},
			{label: "4k", value: "4k"},
		}, "original")
		if promptErr != nil {
			return s, promptErr
		}
		s.resolution, _ = parseResolution(val)
	}
	// non-interactive default: "" (keep original)

	// 3. Frame rate
	if flags.fps != "" {
		s.fps, err = strconv.Atoi(flags.fps)
		if err != nil {
			return s, fmt.Errorf("invalid fps %q: expected 0, 24, 30, or 60", flags.fps)
		}
	} else if interactive {
		val, promptErr := promptChoiceValue(reader, out, "Frame rate", []promptChoice{
			{label: "Keep original", value: "0"},
			{label: "24 fps", value: "24"},
			{label: "30 fps", value: "30"},
			{label: "60 fps", value: "60"},
		}, "0")
		if promptErr != nil {
			return s, promptErr
		}
		s.fps, _ = strconv.Atoi(val)
	}
	// non-interactive default: 0 (keep original)

	// 4. Audio quality
	if flags.audio != "" {
		s.audio, err = parseAudioMode(flags.audio)
		if err != nil {
			return s, err
		}
	} else if interactive {
		val, promptErr := promptChoiceValue(reader, out, "Audio quality", []promptChoice{
			{label: "Keep original", value: "keep"},
			{label: "Low (96 kbps)", value: "low"},
			{label: "Medium (128 kbps)", value: "medium"},
			{label: "High (192 kbps)", value: "high"},
		}, "keep")
		if promptErr != nil {
			return s, promptErr
		}
		s.audio, _ = parseAudioMode(val)
	} else {
		s.audio = domain.AudioKeep
	}

	// 5. Output suffix
	s.suffix, err = resolveSuffix(flags.suffix, reader, out, interactive)
	if err != nil {
		return s, err
	}

	// 6. Skip already converted?
	if flags.skipConverted != "" {
		switch strings.ToLower(flags.skipConverted) {
		case "yes":
			s.skipConverted = true
		case "no":
			s.skipConverted = false
		default:
			return s, fmt.Errorf("invalid --skip-converted %q: expected yes or no", flags.skipConverted)
		}
	} else if interactive {
		val, promptErr := promptChoiceValue(reader, out, "Skip already converted?", []promptChoice{
			{label: "Yes (skip files that already have a converted output)", value: "yes"},
			{label: "No (re-encode everything)", value: "no"},
		}, "yes")
		if promptErr != nil {
			return s, promptErr
		}
		s.skipConverted = val == "yes"
	} else {
		s.skipConverted = true
	}

	return s, nil
}

func resolveSuffix(flagValue string, reader *bufio.Reader, out io.Writer, interactive bool) (string, error) {
	if flagValue != "" {
		return normaliseSuffix(flagValue)
	}
	if interactive {
		return promptSuffix(reader, out, "_compressed")
	}
	return "_compressed", nil
}

func promptSuffix(reader *bufio.Reader, out io.Writer, defaultValue string) (string, error) {
	fmt.Fprintf(out, "Output suffix (default: %s):\n> ", defaultValue)
	line, err := reader.ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return "", err
	}
	answer := strings.TrimSpace(line)
	if answer == "" {
		return defaultValue, nil
	}
	return normaliseSuffix(answer)
}

var validSuffix = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

func normaliseSuffix(s string) (string, error) {
	s = strings.TrimSpace(s)
	if !strings.HasPrefix(s, "_") {
		s = "_" + s
	}
	raw := strings.TrimPrefix(s, "_")
	if !validSuffix.MatchString(raw) {
		return "", fmt.Errorf("invalid suffix %q: use only letters, numbers, underscores, and hyphens", s)
	}
	return s, nil
}

// --- Parsers ---

func parseStrategy(value string) (domain.CompressionStrategy, error) {
	switch domain.CompressionStrategy(strings.ToLower(value)) {
	case domain.StrategyQuality:
		return domain.StrategyQuality, nil
	case domain.StrategyBalanced:
		return domain.StrategyBalanced, nil
	case domain.StrategySizePriority:
		return domain.StrategySizePriority, nil
	default:
		return "", fmt.Errorf("unknown strategy %q (expected quality, balanced, or size)", value)
	}
}

func parseResolution(value string) (domain.Resolution, error) {
	switch strings.ToLower(value) {
	case "original", "keep", "":
		return "", nil // keep original
	case "720p":
		return domain.Res720p, nil
	case "1080p":
		return domain.Res1080p, nil
	case "4k":
		return domain.Res4K, nil
	default:
		return "", fmt.Errorf("unknown resolution %q (expected original, 720p, 1080p, or 4k)", value)
	}
}

func parseAudioMode(value string) (domain.AudioMode, error) {
	switch domain.AudioMode(strings.ToLower(value)) {
	case domain.AudioKeep:
		return domain.AudioKeep, nil
	case domain.AudioLow:
		return domain.AudioLow, nil
	case domain.AudioMedium:
		return domain.AudioMedium, nil
	case domain.AudioHigh:
		return domain.AudioHigh, nil
	default:
		return "", fmt.Errorf("unknown audio mode %q (expected keep, low, medium, or high)", value)
	}
}

// --- Prompts ---

type promptChoice struct {
	label string
	value string
}

func promptChoiceValue(reader *bufio.Reader, out io.Writer, title string, choices []promptChoice, defaultValue string) (string, error) {
	fmt.Fprintf(out, "\n%s:\n", title)
	for i, choice := range choices {
		defaultMarker := ""
		if choice.value == defaultValue {
			defaultMarker = " (default)"
		}
		fmt.Fprintf(out, "  %d. %s%s\n", i+1, choice.label, defaultMarker)
	}

	for {
		fmt.Fprint(out, "> ")
		line, err := reader.ReadString('\n')
		if err != nil && !errors.Is(err, io.EOF) {
			return "", err
		}

		answer := strings.TrimSpace(strings.ToLower(line))
		if answer == "" {
			return defaultValue, nil
		}

		for i, choice := range choices {
			if answer == strconv.Itoa(i+1) || answer == strings.ToLower(choice.value) {
				return choice.value, nil
			}
		}

		if errors.Is(err, io.EOF) {
			return "", io.EOF
		}
		fmt.Fprintf(out, "Invalid choice. Enter 1-%d or a listed value.\n", len(choices))
	}
}

func isInteractiveInput(file *os.File) bool {
	return term.IsTerminal(int(file.Fd()))
}

func usage() {
	fmt.Println(`vc — Video Compressor

Usage:
  vc [compress] <input-dir> [flags]    Batch compress videos
  vc cleanup <input-dir> [flags]       Delete originals and rename converted outputs
  vc assess <input-dir> [flags]        Run codec/CRF test matrix

Compress flags:
  --strategy        quality|balanced|size   Compression strategy (default: balanced)
  --resolution      original|720p|1080p|4k  Target resolution (default: original)
  --fps             0|24|30|60              Frame rate, 0=keep original (default: 0)
  --audio           keep|low|medium|high    Audio quality (default: keep)
  --suffix          STRING                  Output file suffix (default: _compressed)
  --skip-converted  yes|no                  Skip already converted files (default: yes)
  --workers N                               Parallel jobs (default: CPU/2)
  --dry-run                                 Show what would be encoded

Cleanup flags:
  --suffix       STRING                  Suffix used during compression (default: _compressed)

Assess flags:
  --output DIR                           Report output dir (default: ./comparison_reports)
  --workers N                            Parallel jobs (default: 1)`)
}

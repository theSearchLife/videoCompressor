package app

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/theSearchLife/videoCompressor/internal/domain"
	"github.com/theSearchLife/videoCompressor/internal/port"
)

type CleanupService struct {
	scanner port.Scanner
}

func NewCleanupService(scanner port.Scanner) *CleanupService {
	return &CleanupService{
		scanner: scanner,
	}
}

type CleanupOptions struct {
	InputDir string
	Suffix   string
}

type CleanupAction struct {
	OriginalPath  string
	ConvertedPath string
	FinalPath     string
}

func (c *CleanupService) Plan(ctx context.Context, opts CleanupOptions) ([]CleanupAction, error) {
	files, err := c.scanner.Scan(ctx, opts.InputDir)
	if err != nil {
		return nil, fmt.Errorf("scan: %w", err)
	}

	actions := make([]CleanupAction, 0, len(files))
	seenFinal := make(map[string]string) // final path -> first source path

	for _, path := range files {
		if isDerivedOutputPath(path, opts.Suffix) {
			continue
		}

		convertedPath := domain.CompressOutputPath(path, opts.Suffix)
		if _, err := os.Stat(convertedPath); err != nil {
			continue
		}

		finalPath := domain.BaseOutputPath(path)
		if firstSource, ok := seenFinal[finalPath]; ok {
			log.Printf("WARN: %s and %s both target %s during cleanup, skipping %s",
				firstSource, path, filepath.Base(finalPath), path)
			continue
		}
		seenFinal[finalPath] = path

		actions = append(actions, CleanupAction{
			OriginalPath:  path,
			ConvertedPath: convertedPath,
			FinalPath:     finalPath,
		})
	}

	return actions, nil
}

func (c *CleanupService) Run(ctx context.Context, opts CleanupOptions) ([]CleanupAction, error) {
	actions, err := c.Plan(ctx, opts)
	if err != nil {
		return nil, err
	}

	for _, action := range actions {
		if err := os.Remove(action.OriginalPath); err != nil && !os.IsNotExist(err) {
			return nil, fmt.Errorf("delete original %s: %w", action.OriginalPath, err)
		}
		log.Printf("Deleted original: %s", action.OriginalPath)

		if err := os.Rename(action.ConvertedPath, action.FinalPath); err != nil {
			return nil, fmt.Errorf("rename converted %s -> %s: %w", action.ConvertedPath, action.FinalPath, err)
		}
	}

	return actions, nil
}

package fs

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/theSearchLife/videoCompressor/internal/domain"
)

var videoExtensions = map[string]bool{
	".mp4":  true,
	".mkv":  true,
	".avi":  true,
	".mov":  true,
	".wmv":  true,
	".flv":  true,
	".webm": true,
	".m4v":  true,
	".mpg":  true,
	".mpeg": true,
	".3gp":  true,
	".ts":   true,
}

type Scanner struct{}

func NewScanner() *Scanner {
	return &Scanner{}
}

func (s *Scanner) Scan(_ context.Context, root string, recursive bool) ([]string, error) {
	var files []string

	if recursive {
		err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return nil // skip inaccessible entries
			}
			if d.IsDir() {
				return nil
			}
			if isScannableVideo(path) {
				files = append(files, path)
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
	} else {
		entries, err := os.ReadDir(root)
		if err != nil {
			return nil, err
		}
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			path := filepath.Join(root, entry.Name())
			if isScannableVideo(path) {
				files = append(files, path)
			}
		}
	}

	// Sort by size descending — largest first for better parallelism
	sort.Slice(files, func(i, j int) bool {
		si, _ := os.Stat(files[i])
		sj, _ := os.Stat(files[j])
		if si == nil || sj == nil {
			return false
		}
		return si.Size() > sj.Size()
	})

	return files, nil
}

func isVideo(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	return videoExtensions[ext] && !domain.IsOutputFile(path)
}

func isScannableVideo(path string) bool {
	if !isVideo(path) {
		return false
	}

	info, err := os.Stat(path)
	if err != nil {
		return false
	}

	return info.Size() > 0
}

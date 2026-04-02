package domain

import (
	"path/filepath"
	"strings"
)

var videoExtensions = []string{
	".mp4",
	".mkv",
	".avi",
	".mov",
	".wmv",
	".flv",
	".webm",
	".m4v",
	".mpg",
	".mpeg",
	".3gp",
	".ts",
}

func RecognizedVideoExtensions() []string {
	extensions := make([]string, len(videoExtensions))
	copy(extensions, videoExtensions)
	return extensions
}

func IsRecognizedVideoFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	for _, candidate := range videoExtensions {
		if ext == candidate {
			return true
		}
	}
	return false
}

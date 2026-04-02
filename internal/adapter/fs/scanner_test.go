package fs

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestScanRecursesFiltersMixedContentAndCleansTmpFiles(t *testing.T) {
	root := t.TempDir()

	// Build a realistic fixture tree inline — no dependency on testdata/
	dirs := []string{
		"photos",
		"nested/deep",
	}
	for _, d := range dirs {
		if err := os.MkdirAll(filepath.Join(root, d), 0o755); err != nil {
			t.Fatal(err)
		}
	}

	// Scannable videos (non-empty)
	writeFile(t, filepath.Join(root, "top-level.mp4"), 64)
	writeFile(t, filepath.Join(root, "nested", "clip.mov"), 64)

	// Non-video files — must be skipped
	writeFile(t, filepath.Join(root, "photos", "image.jpg"), 12)
	writeFile(t, filepath.Join(root, "notes.txt"), 12)

	// Empty .ts file — must be skipped (zero-byte video)
	writeFile(t, filepath.Join(root, "empty.ts"), 0)

	// Stale tmp files — must be removed
	writeFile(t, filepath.Join(root, "stale-output.mp4.tmp"), 10)
	writeFile(t, filepath.Join(root, "nested", "orphan.tmp"), 10)

	files, err := NewScanner().Scan(context.Background(), root)
	if err != nil {
		t.Fatal(err)
	}

	want := []string{
		filepath.Join(root, "top-level.mp4"),
		filepath.Join(root, "nested", "clip.mov"),
	}
	if len(files) != len(want) {
		t.Fatalf("expected %d scannable videos, got %d: %v", len(want), len(files), files)
	}
	for _, expected := range want {
		if !contains(files, expected) {
			t.Fatalf("expected %q in scanned files: %v", expected, files)
		}
	}

	if _, err := os.Stat(filepath.Join(root, "stale-output.mp4.tmp")); !os.IsNotExist(err) {
		t.Fatalf("expected top-level tmp file to be removed, got err=%v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "nested", "orphan.tmp")); !os.IsNotExist(err) {
		t.Fatalf("expected nested tmp file to be removed, got err=%v", err)
	}
}

func writeFile(t *testing.T, path string, size int) {
	t.Helper()
	data := make([]byte, size)
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
}

func contains(values []string, needle string) bool {
	for _, value := range values {
		if value == needle {
			return true
		}
	}
	return false
}

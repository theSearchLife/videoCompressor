package fs

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestScanSkipsEmptyFiles(t *testing.T) {
	root := t.TempDir()

	if err := os.WriteFile(filepath.Join(root, "valid.mp4"), []byte("data"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "empty.ts"), nil, 0o644); err != nil {
		t.Fatal(err)
	}

	files, err := NewScanner().Scan(context.Background(), root, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 1 {
		t.Fatalf("expected 1 scannable video, got %d: %v", len(files), files)
	}
}

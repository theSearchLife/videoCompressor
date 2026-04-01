package fs

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestScanRecursesFiltersMixedContentAndCleansTmpFiles(t *testing.T) {
	root := copyFixtureTree(t, filepath.Join(projectRoot(t), "testdata", "mixed-content"))
	if err := os.WriteFile(filepath.Join(root, "empty.ts"), nil, 0o644); err != nil {
		t.Fatal(err)
	}

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

func copyFixtureTree(t *testing.T, fixtureRoot string) string {
	t.Helper()

	dstRoot := t.TempDir()
	err := filepath.WalkDir(fixtureRoot, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		rel, err := filepath.Rel(fixtureRoot, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}

		dstPath := filepath.Join(dstRoot, rel)
		if d.IsDir() {
			return os.MkdirAll(dstPath, 0o755)
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(dstPath, data, 0o644)
	})
	if err != nil {
		t.Fatal(err)
	}

	return dstRoot
}

func projectRoot(t *testing.T) string {
	t.Helper()

	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", ".."))
}

func contains(values []string, needle string) bool {
	for _, value := range values {
		if value == needle {
			return true
		}
	}
	return false
}

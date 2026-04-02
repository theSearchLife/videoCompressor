package app

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

type cleanupScanner struct {
	files []string
}

func (c cleanupScanner) Scan(context.Context, string) ([]string, error) {
	return c.files, nil
}

func TestCleanupRenamesConvertedOutputsIntoPlace(t *testing.T) {
	root := t.TempDir()
	original := filepath.Join(root, "clip.mov")
	converted := filepath.Join(root, "clip_compressed.mp4")
	finalPath := filepath.Join(root, "clip.mp4")

	if err := os.WriteFile(original, []byte("source"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(converted, []byte("encoded"), 0o644); err != nil {
		t.Fatal(err)
	}

	service := NewCleanupService(
		cleanupScanner{files: []string{original, converted}},
	)

	actions, err := service.Run(context.Background(), CleanupOptions{
		InputDir: root,
		Suffix:   "_compressed",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(actions) != 1 {
		t.Fatalf("expected 1 cleanup action, got %d", len(actions))
	}

	if _, err := os.Stat(original); !os.IsNotExist(err) {
		t.Fatalf("expected original to be removed, got err=%v", err)
	}
	if _, err := os.Stat(converted); !os.IsNotExist(err) {
		t.Fatalf("expected converted output to be renamed away, got err=%v", err)
	}

	data, err := os.ReadFile(finalPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "encoded" {
		t.Fatalf("expected renamed output contents to survive, got %q", string(data))
	}
}

func TestCleanupDeduplicatesCollidingFinalPaths(t *testing.T) {
	root := t.TempDir()
	clipMov := filepath.Join(root, "clip.mov")
	clipAvi := filepath.Join(root, "clip.avi")
	converted := filepath.Join(root, "clip_compressed.mp4")

	if err := os.WriteFile(clipMov, []byte("source-mov"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(clipAvi, []byte("source-avi"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(converted, []byte("encoded"), 0o644); err != nil {
		t.Fatal(err)
	}

	service := NewCleanupService(
		cleanupScanner{files: []string{clipMov, clipAvi, converted}},
	)

	actions, err := service.Plan(context.Background(), CleanupOptions{
		InputDir: root,
		Suffix:   "_compressed",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(actions) != 1 {
		t.Fatalf("expected 1 cleanup action (collision deduped), got %d", len(actions))
	}
}

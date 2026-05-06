package ffmpeg

import (
	"os"
	"path/filepath"
	"runtime"
)

// resolveBinary returns a path for the named ffmpeg helper that prefers a copy
// living next to the running vc binary, falling back to the bare command name
// so the OS PATH lookup still applies.
//
// This lets the Windows release zip ship vc.exe, ffmpeg.exe and ffprobe.exe in
// the same folder and Just Work without the user editing PATH, while leaving
// Linux/Mac users free to use their package-managed ffmpeg.
func resolveBinary(name string) string {
	exe, err := os.Executable()
	if err != nil {
		return name
	}
	dir := filepath.Dir(exe)

	candidates := []string{filepath.Join(dir, name)}
	if runtime.GOOS == "windows" {
		candidates = append(candidates, filepath.Join(dir, name+".exe"))
	}
	for _, c := range candidates {
		if info, err := os.Stat(c); err == nil && !info.IsDir() {
			return c
		}
	}
	return name
}

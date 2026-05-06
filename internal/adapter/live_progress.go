package adapter

import (
	"fmt"
	"io"
	"os"
	"sync"

	"golang.org/x/term"
)

// liveProgress prints one updating-in-place line per active encode job.
// Regular log lines (START / DONE / WARN / Summary) scroll above the live
// region. Implementation uses bare ANSI cursor moves (CSI A / CSI J) — these
// work in Linux/macOS terminals and in modern Windows Terminal, including
// Docker Desktop's PowerShell when Docker is run with -it.
//
// When stdout is not a TTY the live region is suppressed entirely and only
// log lines are emitted, so output stays clean for piped/captured runs.
type liveProgress struct {
	out   io.Writer
	tty   bool
	mu    sync.Mutex
	lines []*liveLine
	drawn int
}

type liveLine struct {
	id   int
	text string
}

func newLiveProgress(out io.Writer) *liveProgress {
	tty := false
	if f, ok := out.(*os.File); ok && f != nil {
		tty = term.IsTerminal(int(f.Fd()))
	}
	return &liveProgress{out: out, tty: tty}
}

// Write implements io.Writer. Go's log package is reconfigured in main to
// route every log line through here so it lands above the live region.
func (p *liveProgress) Write(b []byte) (int, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.clearRegion()
	n, err := p.out.Write(b)
	p.drawRegion()
	return n, err
}

// addLine appends a new live row for the given job ID with starting text.
func (p *liveProgress) addLine(id int, text string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	for _, ln := range p.lines {
		if ln.id == id {
			ln.text = text
			p.refresh()
			return
		}
	}
	p.lines = append(p.lines, &liveLine{id: id, text: text})
	p.refresh()
}

// updateLine replaces the rendered text for an existing live row.
func (p *liveProgress) updateLine(id int, text string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	for _, ln := range p.lines {
		if ln.id == id {
			if ln.text == text {
				return
			}
			ln.text = text
			p.refresh()
			return
		}
	}
}

// removeLine drops a live row; the final DONE/FAIL log line is normally
// emitted just before this call so the user sees the outcome scroll above.
func (p *liveProgress) removeLine(id int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	for i, ln := range p.lines {
		if ln.id == id {
			p.lines = append(p.lines[:i], p.lines[i+1:]...)
			p.refresh()
			return
		}
	}
}

func (p *liveProgress) refresh() {
	p.clearRegion()
	p.drawRegion()
}

func (p *liveProgress) clearRegion() {
	if !p.tty || p.drawn == 0 {
		return
	}
	// CSI nA: cursor up n lines. CSI J: erase from cursor to end of screen.
	fmt.Fprintf(p.out, "\r\033[%dA\033[J", p.drawn)
	p.drawn = 0
}

func (p *liveProgress) drawRegion() {
	if !p.tty || len(p.lines) == 0 {
		return
	}
	for _, ln := range p.lines {
		fmt.Fprintln(p.out, ln.text)
	}
	p.drawn = len(p.lines)
}

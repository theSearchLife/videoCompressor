package adapter

import (
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"sync"
	"unicode/utf8"

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
	out           io.Writer
	tty           bool
	terminalWidth func() int
	mu            sync.Mutex
	lines         []*liveLine
	drawn         int
}

type liveLine struct {
	id   int
	text string
}

func newLiveProgress(out io.Writer) *liveProgress {
	tty := false
	var terminalWidth func() int
	if f, ok := out.(*os.File); ok && f != nil {
		tty = term.IsTerminal(int(f.Fd()))
		terminalWidth = func() int {
			width, _, err := term.GetSize(int(f.Fd()))
			if err != nil {
				return terminalWidthFromEnv()
			}
			return width
		}
	}
	return &liveProgress{out: out, tty: tty, terminalWidth: terminalWidth}
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
	width := p.lineWidth()
	for _, ln := range p.lines {
		fmt.Fprintln(p.out, fitLiveText(ln.text, width))
	}
	p.drawn = len(p.lines)
}

func (p *liveProgress) lineWidth() int {
	if !p.tty {
		return 0
	}
	width := 0
	if p.terminalWidth != nil {
		width = p.terminalWidth()
	}
	if width <= 1 {
		width = terminalWidthFromEnv()
	}
	if width <= 1 {
		width = 80
	}
	// Leave one column spare so terminals do not auto-wrap before the newline.
	return width - 1
}

func terminalWidthFromEnv() int {
	width, err := strconv.Atoi(os.Getenv("COLUMNS"))
	if err != nil {
		return 0
	}
	return width
}

func fitLiveText(text string, maxWidth int) string {
	text = singleLineText(text)
	if maxWidth <= 0 || textWidth(text) <= maxWidth {
		return text
	}
	return truncateEnd(text, maxWidth)
}

func compactMiddle(text string, maxWidth int) string {
	text = singleLineText(text)
	if maxWidth <= 0 {
		return ""
	}
	if textWidth(text) <= maxWidth {
		return text
	}
	if maxWidth <= 3 {
		return strings.Repeat(".", maxWidth)
	}

	available := maxWidth - 3
	headWidth := available / 2
	tailWidth := available - headWidth
	return takeStart(text, headWidth) + "..." + takeEnd(text, tailWidth)
}

func truncateEnd(text string, maxWidth int) string {
	if maxWidth <= 0 {
		return ""
	}
	if maxWidth <= 3 {
		return strings.Repeat(".", maxWidth)
	}
	return takeStart(text, maxWidth-3) + "..."
}

func takeStart(text string, maxWidth int) string {
	if maxWidth <= 0 {
		return ""
	}
	width := 0
	for i, r := range text {
		next := width + runeWidth(r)
		if next > maxWidth {
			return text[:i]
		}
		width = next
	}
	return text
}

func takeEnd(text string, maxWidth int) string {
	if maxWidth <= 0 {
		return ""
	}
	width := 0
	end := len(text)
	for end > 0 {
		r, size := utf8.DecodeLastRuneInString(text[:end])
		next := width + runeWidth(r)
		if next > maxWidth {
			return text[end:]
		}
		width = next
		end -= size
	}
	return text
}

func singleLineText(text string) string {
	return strings.Map(func(r rune) rune {
		switch r {
		case '\r', '\n', '\t':
			return ' '
		default:
			return r
		}
	}, text)
}

func textWidth(text string) int {
	width := 0
	for _, r := range text {
		width += runeWidth(r)
	}
	return width
}

func runeWidth(r rune) int {
	switch {
	case r < 32 || r == 127:
		return 0
	case r < utf8.RuneSelf:
		return 1
	default:
		// Conservative for non-ASCII filenames: some terminals render these as
		// double-width, and shorter is safer than triggering a wrap.
		return 2
	}
}

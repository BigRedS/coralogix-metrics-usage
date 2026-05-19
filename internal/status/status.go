// Package status provides a single-line progress display on stderr.
package status

import (
	"fmt"
	"io"
	"os"
	"strings"
	"sync"

	"golang.org/x/term"
)

// Line writes one status message, overwriting the previous line when stderr is a TTY.
type Line struct {
	w           io.Writer
	mu          sync.Mutex
	interactive bool
	width       int
}

// New creates a status line writing to w (typically os.Stderr).
func New(w io.Writer) *Line {
	interactive := false
	width := 100
	if f, ok := w.(*os.File); ok {
		interactive = term.IsTerminal(int(f.Fd()))
		if interactive {
			if tw, _, err := term.GetSize(int(f.Fd())); err == nil && tw > 20 {
				width = tw
			}
		}
	}
	return &Line{w: w, interactive: interactive, width: width}
}

// Set updates the current status text.
func (l *Line) Set(msg string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if !l.interactive {
		fmt.Fprintln(l.w, msg)
		return
	}
	if len(msg) > l.width-1 {
		msg = msg[:l.width-4] + "..."
	}
	pad := l.width - len(msg)
	if pad < 0 {
		pad = 0
	}
	fmt.Fprintf(l.w, "\r%s%s", msg, strings.Repeat(" ", pad))
}

// Done clears the in-progress line and optionally prints a final message.
func (l *Line) Done(final string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.interactive {
		fmt.Fprintf(l.w, "\r%s\r\n", strings.Repeat(" ", l.width))
	}
	if final != "" {
		fmt.Fprintln(l.w, final)
	}
}

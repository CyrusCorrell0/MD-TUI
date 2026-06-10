package main

import (
	"log"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/CyrusCorrell0/MD-TUI/model"
)

// syncWriter wraps stdout so that every Bubble Tea renderer flush (one Write per
// frame) is bracketed in DEC private mode 2026 "synchronized output" markers. The
// terminal buffers the whole frame and presents it atomically, eliminating the
// erase-line/repaint flicker the standard renderer would otherwise show during
// full-canvas repaints (playback, rotation). Terminals without 2026 support simply
// ignore the unknown private-mode sequences.
//
// It embeds *os.File so it still satisfies bubbletea's term.File check (Fd()). That
// is required on Windows to enable virtual-terminal processing and to read the
// terminal size; a plain io.Writer would break both.
type syncWriter struct {
	*os.File
}

const (
	beginSync = "\x1b[?2026h"
	endSync   = "\x1b[?2026l"
)

func (w syncWriter) Write(p []byte) (int, error) {
	buf := make([]byte, 0, len(p)+len(beginSync)+len(endSync))
	buf = append(buf, beginSync...)
	buf = append(buf, p...)
	buf = append(buf, endSync...)
	if _, err := w.File.Write(buf); err != nil {
		return 0, err
	}
	return len(p), nil
}

func main() {
	p := tea.NewProgram(
		model.NewAppModel(),
		tea.WithAltScreen(),
		tea.WithOutput(syncWriter{os.Stdout}),
	)
	if _, err := p.Run(); err != nil {
		log.Fatalf("failed to run app: %v", err)
	}
}

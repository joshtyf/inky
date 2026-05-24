package terminal

import (
	"context"
	"fmt"
	"os"
	"strings"
	"unicode/utf8"

	"github.com/joshtyf/inky/core"
	"golang.org/x/sys/unix"
	"golang.org/x/term"
)

var keyMapping = map[string]core.Key{
	// Escape Sequences
	"\x1b[A":    {Code: core.ArrowUp},
	"\x1b[B":    {Code: core.ArrowDown},
	"\x1b[D":    {Code: core.ArrowLeft},
	"\x1b[C":    {Code: core.ArrowRight},
	"\x1b[1;2A": {Code: core.ShiftArrowUp},
	"\x1b[1;2B": {Code: core.ShiftArrowDown},
	"\x1b[1;2C": {Code: core.ShiftArrowRight},
	"\x1b[1;2D": {Code: core.ShiftArrowLeft},
	"\x1b":      {Code: core.ToggleViewMode},

	// Control Keys
	"\x04": {Code: core.CtrlD},
	"\x7f": {Code: core.Backspace},
	"\x1f": {Code: core.Undo},
	"\x13": {Code: core.Save},
}

type Renderer interface {
	Render(line int, es *core.EditorState) (string, error)
	LastLine(es *core.EditorState) int
}

var renderers = map[core.ViewMode]Renderer{
	core.MarkdownView: NewMarkdownRenderer(),
	core.RawView:      NewRawRenderer(),
}

type Terminal struct {
	topLine                 int
	currentLine             int
	screenHeight            int
	renderCache             []string
	markdownViewCurrentLine int
	lastContentVersion      int
}

func NewTerminal() *Terminal {
	_, h, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil {
		panic("error getting terminal size: " + err.Error())
	}
	return &Terminal{
		topLine:                 0,
		currentLine:             0,
		screenHeight:            h,
		renderCache:             make([]string, h-1),
		markdownViewCurrentLine: 0,
		lastContentVersion:      -1,
	}
}

func (t *Terminal) Init() error {
	termios, err := unix.IoctlGetTermios(int(os.Stdin.Fd()), unix.TIOCGETA)
	if err != nil {
		return fmt.Errorf("error getting stdin terminal attributes: %w", err)
	}
	termios.Lflag &^= unix.ICANON | unix.ECHO
	termios.Iflag &^= unix.IXON // Disable flow control for Ctrl+S
	err = unix.IoctlSetTermios(int(os.Stdin.Fd()), unix.TIOCSETA, termios)
	if err != nil {
		return fmt.Errorf("error setting stdin terminal attributes: %w", err)
	}
	fmt.Print(AnsiClearScreen)
	fmt.Print(AnsiCursorHome)
	return nil
}

func (t *Terminal) Close() error {
	termios, err := unix.IoctlGetTermios(int(os.Stdin.Fd()), unix.TIOCGETA)
	if err != nil {
		return fmt.Errorf("error getting stdin terminal attributes: %w", err)
	}
	termios.Lflag &^= unix.ICANON | unix.ECHO
	termios.Iflag &^= unix.IXON // Restore flow control for Ctrl+S
	err = unix.IoctlSetTermios(int(os.Stdin.Fd()), unix.TIOCSETA, termios)
	if err != nil {
		return fmt.Errorf("error resetting stdin terminal attributes: %w", err)
	}
	fmt.Print(AnsiCursorShow)
	return nil
}

func (t *Terminal) parseSpecialSequences(b []byte) (core.Key, int) {
	var matchedKey core.Key
	matchedKeySize := 0
	for k, v := range keyMapping {
		predicateLen := len(k)
		if len(b) < predicateLen {
			continue
		}
		// Longest prefix match
		if string(b[:predicateLen]) == k && matchedKeySize < predicateLen {
			matchedKey = v
			matchedKeySize = predicateLen
		}
	}
	return matchedKey, matchedKeySize
}

func (t *Terminal) GetKey(ctx context.Context) <-chan core.Key {
	byteCh := make(chan []byte)
	go func() {
		defer close(byteCh)
		for {
			const INPUT_BUFFER_SIZE = 256
			var b [INPUT_BUFFER_SIZE]byte
			n, err := os.Stdin.Read(b[:])
			if err != nil {
				return // Graceful exit on read error (e.g., EOF or closed stdin)
			}
			if n > 0 {
				chunk := make([]byte, n)
				copy(chunk, b[:n])
				byteCh <- chunk
			}
		}
	}()

	ch := make(chan core.Key)
	go func() {
		defer close(ch)
		var buf []byte
		for {
			select {
			case <-ctx.Done():
				return
			case chunk, ok := <-byteCh:
				if !ok {
					return
				}
				buf = append(buf, chunk...)

				for len(buf) > 0 {
					k, size := t.parseSpecialSequences(buf)
					if size > 0 {
						ch <- k
						buf = buf[size:]
						continue
					}

					if !utf8.FullRune(buf) {
						break // Wait for more bytes to complete the rune
					}

					r, size := utf8.DecodeRune(buf)
					if r == utf8.RuneError {
						// Skip invalid byte instead of panicking
						buf = buf[1:]
						continue
					}

					ch <- core.Key{Code: core.RuneKey, Rune: r}
					buf = buf[size:]
				}
			}
		}
	}()
	return ch
}

func (t *Terminal) Update(es *core.EditorState) error {
	if es.Version != t.lastContentVersion {
		t.markdownViewCurrentLine = 0
		t.lastContentVersion = es.Version
	}
	if es.ViewMode == core.MarkdownView && es.LastKeyPresssed != nil {
		switch es.LastKeyPresssed.Code {
		case core.ArrowDown:
			if t.markdownViewCurrentLine < renderers[core.MarkdownView].LastLine(es) {
				t.markdownViewCurrentLine++
			}
		case core.ArrowUp:
			if t.markdownViewCurrentLine > 0 {
				t.markdownViewCurrentLine--
			}
		}
	}
	var targetLine, targetCol int
	if es.ViewMode == core.MarkdownView {
		targetLine = t.markdownViewCurrentLine
		targetCol = 1
	} else {
		targetLine = es.CursorCurrentLine
		targetCol = es.CursorCurrentCol + 1
	}
	// Update topLine for scrolling before rendering
	if targetLine >= t.topLine+t.screenHeight-1 {
		t.topLine = targetLine - t.screenHeight + 2
	} else if targetLine < t.topLine {
		t.topLine = targetLine
	}
	fmt.Print(AnsiCursorHide)
	t.updateTextCache(es)
	t.renderText()
	t.renderUI(es)
	// Reposition cursor explicitly after all rendering
	fmt.Print(AnsiMoveCursor(targetLine-t.topLine+1, targetCol))
	fmt.Print(AnsiCursorShow)
	return nil
}

func (t *Terminal) updateTextCache(es *core.EditorState) {
	renderer := renderers[es.ViewMode]
	for i := 0; i < t.screenHeight-1; i++ {
		lineNum := t.topLine + i
		line, err := renderer.Render(lineNum, es)
		if err != nil {
			panic("error rendering line: " + err.Error())
		}
		t.renderCache[i] = line
	}
}

func (t *Terminal) renderText() {
	for i := range t.renderCache {
		fmt.Print(AnsiMoveToLineStart(i + 1))
		fmt.Print(AnsiEraseLine)
		fmt.Print(t.renderCache[i])
	}
}

func (t *Terminal) renderUI(es *core.EditorState) {
	viewMode := "Raw"
	if es.ViewMode == core.MarkdownView {
		viewMode = "Markdown"
	}
	status := fmt.Sprintf("Line: %d, Col: %d, Mode: %s", es.CursorCurrentLine+1, es.CursorCurrentCol+1, viewMode)
	if es.ViewMode == core.MarkdownView {
		cacheIdx := t.markdownViewCurrentLine - t.topLine
		if cacheIdx >= 0 && cacheIdx < t.screenHeight-1 {
			fmt.Print(AnsiMoveToLineStart(cacheIdx + 1))
			fmt.Print(AnsiEraseLine)
			fmt.Print(highlightLine(t.renderCache[cacheIdx]))
		}
	}
	fmt.Print(AnsiMoveToLineStart(t.screenHeight))
	fmt.Print(AnsiInverse)
	fmt.Print(AnsiEraseLine)
	fmt.Print(status)
	fmt.Print(AnsiReset)
}

func highlightLine(line string) string {
	return AnsiInverse + strings.ReplaceAll(line, AnsiReset, AnsiReset+AnsiInverse) + AnsiReset
}

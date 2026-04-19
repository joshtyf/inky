package ui

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"os"
	"strings"
	"unicode/utf8"

	"github.com/joshtyf/inky/core"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/util"
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
	"\x1b":      {Code: core.Escape},

	// Control Keys
	"\x04": {Code: core.CtrlD},
	"\x7f": {Code: core.Backspace},
	"\x1f": {Code: core.Undo},
	"\x13": {Code: core.Save},
}

const (
	MarkdownView = iota
	RawView
)

type Terminal struct {
	topLine      int
	currentLine  int
	screenHeight int
	mdRenderer   goldmark.Markdown
	viewMode     int
	renderCache  []string
}

type TerminalMdRenderer struct{}

func NewTerminalMdRenderer() *TerminalMdRenderer {
	return &TerminalMdRenderer{}
}

func (t *TerminalMdRenderer) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	reg.Register(ast.KindParagraph, func(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			_, err := w.WriteString("\n\n")
			if err != nil {
				panic(err)
			}
		}
		return ast.WalkContinue, nil
	})
	reg.Register(ast.KindEmphasis, func(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
		n := node.(*ast.Emphasis)
		if entering {
			tag := "\x1b[3m" // Italic
			if n.Level == 2 {
				tag = "\x1b[1m" // Bold
			}
			_, err := w.WriteString(tag)
			if err != nil {
				panic(err)
			}
		} else {
			_, err := w.WriteString("\x1b[0m") // Reset
			if err != nil {
				panic(err)
			}
		}
		return ast.WalkContinue, nil
	})
	reg.Register(ast.KindText, func(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		n := node.(*ast.Text)
		_, err := w.Write(n.Segment.Value(source))
		return ast.WalkContinue, err
	})
}
func NewTerminal() *Terminal {
	_, h, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil {
		panic("error getting terminal size: " + err.Error())
	}
	return &Terminal{
		topLine:      0,
		currentLine:  0,
		screenHeight: h,
		mdRenderer: goldmark.New(
			goldmark.WithRenderer(
				renderer.NewRenderer(
					renderer.WithNodeRenderers(util.Prioritized(NewTerminalMdRenderer(), 100)),
				),
			),
		),
		viewMode:    RawView,
		renderCache: make([]string, h),
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
	fmt.Print("\x1b[2J") // Clear the screen
	fmt.Print("\x1b[H")  // Move cursor to top-left corner
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
	return nil
}

func (t *Terminal) parseSpecialSequences(b []byte) (*core.Key, int) {
	bufstr := string(b)
	var matchedKey *core.Key
	matchedKeySize := 0
	for k, v := range keyMapping {
		predicateLen := len(k)
		if len(bufstr) < predicateLen {
			continue
		}
		// Longest prefix match
		if strings.HasPrefix(bufstr, k) && matchedKeySize < predicateLen {
			matchedKey = &v
			matchedKeySize = predicateLen
		}
	}
	return matchedKey, matchedKeySize
}

func (t *Terminal) GetKey(ctx context.Context) <-chan *core.Key {
	ch := make(chan *core.Key)
	go func() {
		defer close(ch)
		for {
			select {
			case <-ctx.Done():
				return
			default:
				const INPUT_BUFFER_SIZE = 256
				var b [INPUT_BUFFER_SIZE]byte
				n, err := os.Stdin.Read(b[:])
				if err != nil {
					panic("input failed: " + err.Error())
				}
				for i := 0; i < n; i++ {
					k, size := t.parseSpecialSequences(b[i:])
					if k != nil {
						ch <- k
						i += size - 1 // Move to the next character after the escape sequence
						continue
					}

					// If we reach here, then we try to decode the bytes as a rune
					r, size := utf8.DecodeRune(b[i:])
					if r == utf8.RuneError {
						if size == 1 {
							panic(fmt.Sprintf("error decoding rune: invalid byte sequence %v", b[i:]))
						} else {
							panic("error decoding rune: empty byte sequence")
						}
					}
					ch <- &core.Key{Code: core.RuneKey, Rune: r}
					i += size - 1 // Move to the next sequence
				}
			}
		}
	}()
	return ch
}

func (t *Terminal) Update(es *core.EditorState) error {
	t.moveCursor(es.CurrentLine+1, es.CurrentCol+1) // Add 1 because terminal escape codes are 1-indexed
	if es.LastKeyPresssed.Code == core.Escape {
		t.viewMode = (t.viewMode + 1) % 2
	}
	t.updateRenderCache(es)
	t.render()
	t.renderStatusLine(es)
	return nil
}

func (t *Terminal) moveCursor(line, column int) {
	if line >= t.topLine+t.screenHeight {
		t.topLine = line - t.screenHeight + 1
	} else if line < t.topLine {
		t.topLine = line
	}
	fmt.Printf("\x1b[%d;%dH", line-t.topLine, column)
}

func (t *Terminal) updateRenderCache(es *core.EditorState) {
	switch t.viewMode {
	case MarkdownView:
		var buf bytes.Buffer
		source := es.GetAll()
		if err := t.mdRenderer.Convert(source, &buf); err != nil {
			panic(err)
		}
		scanner := bufio.NewScanner(&buf)
		for i := 0; i < t.screenHeight; i++ {
			if scanner.Scan() {
				t.renderCache[i] = scanner.Text()
			} else {
				t.renderCache[i] = ""
			}
		}
	case RawView:
		for i := 0; i < t.screenHeight; i++ {
			t.renderCache[i] = string(es.GetLine(t.topLine + i))
		}
	default:
		panic("invalid view mode")
	}
}

func (t *Terminal) render() {
	fmt.Print("\x1b[s") // Save cursor
	for i := range t.renderCache {
		fmt.Printf("\x1b[%d;1H\x1b[2K%s", i+1, t.renderCache[i]) // Move to the beginning of the line, clear it, and print the new content
	}
	fmt.Print("\x1b[u") // Restore cursor
}

func (t *Terminal) renderStatusLine(es *core.EditorState) {
	viewMode := "Raw"
	if t.viewMode == MarkdownView {
		viewMode = "Markdown"
	}
	status := fmt.Sprintf("Line: %d, Col: %d, Mode: %s", es.CurrentLine+1, es.CurrentCol+1, viewMode)
	fmt.Print("\x1b[s") // Save cursor
	// Highlight the status line with inverse colors
	fmt.Printf("\x1b[%d;1H\x1b[7m\x1b[2K%s\x1b[0m", t.screenHeight, status) // Move to the last line, clear it, and print the status
	fmt.Print("\x1b[u")                                                     // Restore cursor
}

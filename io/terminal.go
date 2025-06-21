package io

import (
	"bytes"
	"errors"
	"fmt"
	"log"
	"os"
	"unicode/utf8"

	"github.com/joshtyf/texteditor/core"
	editorLog "github.com/joshtyf/texteditor/log"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/util"
	"golang.org/x/sys/unix"
	"golang.org/x/term"
)

const (
	clearScreen    = "\033[2J"
	cursorHome     = "\033[H"
	highlightStart = "\033[7m"
	highlightEnd   = "\033[0m"
	boldStart      = "\033[1m"
	boldEnd        = "\033[22m"
	italicStart    = "\033[3m"
	italicEnd      = "\033[23m"
)

const DEFAULT_BUFFER_READ_SIZE = 256

var defaultMapping = map[string]core.Key{
	// Arrow Keys
	"\x1b[A": {Code: core.ArrowUp},
	"\x1b[B": {Code: core.ArrowDown},
	"\x1b[D": {Code: core.ArrowLeft},
	"\x1b[C": {Code: core.ArrowRight},

	// Control Keys
	"\x04": {Code: core.CtrlD},
	"\x7f": {Code: core.Backspace},
	"\x1f": {Code: core.Undo},
	"\x0a": {Code: core.Newline},
}

type TerminalIO struct {
	top            int
	mapping        map[string]core.Key
	logger         *log.Logger
	outputRenderer goldmark.Markdown
}

func NewTerminalIO() *TerminalIO {
	// TODO: allow for custom sequences
	return &TerminalIO{
		top:     0,
		mapping: defaultMapping,
		logger:  editorLog.CreateLogger("terminalIO"),
		outputRenderer: goldmark.New(
			goldmark.WithRenderer(
				renderer.NewRenderer(
					renderer.WithNodeRenderers(
						util.Prioritized(NewTerminalRenderer(), 1000),
					),
				),
			),
		),
	}
}

// Indicate to user that teardown must be called as defer
func (t *TerminalIO) StartIO() (<-chan *core.Key, error) {
	t.setup()
	inputCh := make(chan *core.Key)
	go func() {
		defer close(inputCh)

		for {
			k, err := t.listen()
			if err != nil {
				t.logger.Printf("error listening for input: %v", err)
				return
			}
			inputCh <- k
		}
	}()
	return inputCh, nil
}

func (t *TerminalIO) SetDisplay(es *core.EditorState) error {
	_, h, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil {
		t.logger.Printf("error getting terminal size: %v", err)
	}
	// Status line
	if es.Error != nil {
		// Last line reserved for status line
		// TODO: refactor updating of status line into a separate function
		t.writeLine(h-1, fmt.Sprintf("~error~ %s", es.Error.Error()))
		return nil
	}
	fmt.Printf("%s%s", cursorHome, clearScreen)
	// Reposition screen to match editor view
	if es.CurrentLine < t.top {
		t.top = es.CurrentLine
	} else if es.CurrentLine >= t.top+h-1 {
		t.top = es.CurrentLine - h + 2
	}
	content, err := es.ReadEditorLines(t.top, h-1) // Last line reserved for status line
	if err != nil {
		// TODO: update the status line? how to handle this?
		t.logger.Printf("error reading lines: %v", err)
	}
	for i := range content {
		// TODO: Use goldmark and create your own terminal renderer
		var renderedContent bytes.Buffer
		if err := t.outputRenderer.Convert(content[i], &renderedContent); err != nil {
			panic(err)
		}

		t.writeLine(i, fmt.Sprintf("~ %s", &renderedContent))
	}
	t.writeLine(h-1, "~end~")

	// // Reposition cursor to current line and column
	fmt.Printf("\033[%d;%dH", es.CurrentLine-t.top+1, es.CurrentColumn+3)
	return nil
}

func (t *TerminalIO) setup() error {
	if err := t.setupStdin(); err != nil {
		return fmt.Errorf("error setting up stdin: %w", err)
	}
	if err := t.setupStdout(); err != nil {
		return fmt.Errorf("error setting up stdout: %w", err)
	}
	t.logger.Println("terminal input/output setup complete")
	return nil
}

func (t *TerminalIO) setupStdin() error {
	// TODO: Make it cross-platform
	termios, err := unix.IoctlGetTermios(int(os.Stdin.Fd()), unix.TIOCGETA)
	if err != nil {
		t.logger.Println("error getting terminal attributes")
		return err
	}
	termios.Lflag &^= unix.ICANON | unix.ECHO
	t.logger.Printf("setting terminal attributes: %v", termios.Lflag)
	err = unix.IoctlSetTermios(int(os.Stdin.Fd()), unix.TIOCSETA, termios)
	if err != nil {
		log.Println("error setting terminal attributes")
		return err
	}
	return nil
}

func (t *TerminalIO) setupStdout() error {
	_, h, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil {
		return err
	}
	fmt.Printf("%s%s", cursorHome, clearScreen)
	for i := range h {
		if err := t.writeLine(i, "~ "); err != nil {
			return err
		}
	}
	t.writeLine(h-1, "~end~")
	fmt.Print("\033[1;3H")
	return nil
}

func (ot *TerminalIO) writeLine(line int, content string) error {
	_, h, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil {
		return fmt.Errorf("error getting terminal size: %w", err)
	}
	if line >= h {
		return fmt.Errorf("line %d is out of bounds for terminal height %d", line, h)
	}
	os.Stdout.WriteString("\033[s")
	os.Stdout.WriteString(fmt.Sprintf("\033[%d;1H%s", line+1, content))
	os.Stdout.WriteString("\033[u")
	return nil
}

func (t *TerminalIO) Close() error {
	t.logger.Println("closing terminal input/output")
	if err := t.teardownStdin(); err != nil {
		return fmt.Errorf("error tearing down stdin: %w", err)
	}
	if err := t.teardownStdout(); err != nil {
		return fmt.Errorf("error tearing down stdout: %w", err)
	}
	t.logger.Println("terminal input/output closed")
	return nil
}

func (t *TerminalIO) teardownStdin() error {
	t.logger.Println("stopping input")
	termios, err := unix.IoctlGetTermios(int(os.Stdin.Fd()), unix.TIOCGETA)
	if err != nil {
		t.logger.Println("error getting terminal attributes")
	}
	termios.Lflag |= unix.ICANON | unix.ECHO
	err = unix.IoctlSetTermios(int(os.Stdin.Fd()), unix.TIOCSETA, termios)
	if err != nil {
		t.logger.Printf("error resetting terminal attributes: %v", err)
	}
	return nil
}

func (t *TerminalIO) teardownStdout() error {
	t.logger.Println("stopping output")
	fmt.Printf("%s%s", cursorHome, clearScreen)
	return nil
}

func (t *TerminalIO) listen() (*core.Key, error) {
	// Read from stdin
	var b [DEFAULT_BUFFER_READ_SIZE]byte
	n, err := os.Stdin.Read(b[:])
	if err != nil {
		t.logger.Printf("error reading from stdin: %v", err)
		return nil, err
	}

	if k, ok := t.mapping[string(b[:n])]; ok {
		return &k, nil
	}

	runes := make([]rune, 0)
	for i := 0; i < n; i++ {
		r, width := utf8.DecodeRune(b[i:])
		if r == utf8.RuneError {
			t.logger.Printf("error decoding rune: %v", b)
		}
		runes = append(runes, r)
		i += width - 1
	}

	if len(runes) > 0 {
		return &core.Key{Code: core.RuneKey, Runes: runes}, nil
	}
	return nil, errors.New("no key found")
}

type TerminalRenderer struct {
	logger *log.Logger
}

func NewTerminalRenderer() *TerminalRenderer {
	return &TerminalRenderer{
		logger: editorLog.CreateLogger("terminalRenderer"),
	}
}

func (t *TerminalRenderer) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	reg.Register(ast.KindEmphasis, t.renderEmphasis)
	reg.Register(ast.KindText, t.renderText)
}

func (t *TerminalRenderer) renderEmphasis(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	n := node.(*ast.Emphasis)
	if entering {
		if n.Level == 1 {
			_, _ = w.WriteString("*" + italicStart)
		} else {
			_, _ = w.WriteString("**" + boldStart)
		}
	} else {
		if n.Level == 1 {
			_, _ = w.WriteString(italicEnd + "*")
		} else {
			_, _ = w.WriteString(boldEnd + "**")
		}
	}
	return ast.WalkContinue, nil
}

func (t *TerminalRenderer) renderText(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if !entering {
		return ast.WalkContinue, nil
	}
	n := node.(*ast.Text)
	w.Write(n.Segment.Value(source))
	return ast.WalkContinue, nil
}

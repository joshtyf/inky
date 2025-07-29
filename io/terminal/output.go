package terminal

import (
	"bytes"
	"fmt"
	"log"
	"os"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/util"
	"golang.org/x/term"
)

const (
	clearScreen     = "\033[2J"
	cursorHome      = "\033[H"
	highlightStart  = "\033[7m"
	highlightEnd    = "\033[0m"
	boldStart       = "\033[1m"
	boldEnd         = "\033[22m"
	italicStart     = "\033[3m"
	italicEnd       = "\033[23m"
	saveCursor      = "\033[s"
	restoreCursor   = "\033[u"
	eraseEntireLine = "\033[2K"
)

func getGoToLineEscapeSequence(line int) string {
	return fmt.Sprintf("\033[%d;1H", line+1)
}

type output struct {
	logger     *log.Logger
	mdRenderer goldmark.Markdown
}

func newOutput(l *log.Logger) *output {
	return &output{
		logger: l,
		mdRenderer: goldmark.New(
			goldmark.WithRenderer(
				renderer.NewRenderer(
					renderer.WithNodeRenderers(
						util.Prioritized(newTerminalMarkdownRenderer(), 1000),
					),
				),
			),
		),
	}
}

func (o *output) setup() error {
	_, h, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil {
		return err
	}
	_ = o.clearScreen()
	for i := range h {
		if err := o.writeLine(i, []byte{}); err != nil {
			return err
		}
	}
	_ = o.moveCursorToHome()
	return nil
}

func (o *output) reset() error {
	o.clearScreen()
	return nil
}

func (o *output) clearScreen() error {
	fmt.Printf("%s%s", cursorHome, clearScreen)
	return nil
}

func (o *output) moveCursorToHome() error {
	fmt.Print("\033[1;3H") // Default to line 1, column 3
	return nil
}

func (o *output) moveCursor(line, column int) error {
	fmt.Printf("\033[%d;%dH", line+1, column+3)
	return nil
}

func (o *output) writeLine(line int, content []byte) error {
	_, h, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil {
		return fmt.Errorf("error getting terminal size: %w", err)
	}
	if line >= h {
		return fmt.Errorf("line %d is out of bounds for terminal height %d", line, h)
	}
	var renderedContent bytes.Buffer
	if err := o.mdRenderer.Convert(content, &renderedContent); err != nil {
		return fmt.Errorf("error rendering markdown from content: %w", err)
	}
	fmt.Print(saveCursor)
	fmt.Print(getGoToLineEscapeSequence(line))
	fmt.Print(eraseEntireLine)
	fmt.Print("~ ")
	_, err = renderedContent.WriteTo(os.Stdout)
	if err != nil {
		return fmt.Errorf("error writing rendered content to stdout: %w", err)
	}
	fmt.Print(restoreCursor)
	return nil
}

func (o *output) writeRawLine(line int, content string) error {
	_, h, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil {
		return fmt.Errorf("error getting terminal size: %w", err)
	}
	if line >= h {
		return fmt.Errorf("line %d is out of bounds for terminal height %d", line, h)
	}
	fmt.Print(saveCursor)
	fmt.Print(getGoToLineEscapeSequence(line))
	fmt.Print(eraseEntireLine)
	fmt.Print(content)
	fmt.Print(restoreCursor)
	return nil
}

type terminalMarkdownRenderer struct{}

func newTerminalMarkdownRenderer() *terminalMarkdownRenderer {
	return &terminalMarkdownRenderer{}
}

func (t *terminalMarkdownRenderer) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	reg.Register(ast.KindEmphasis, t.renderEmphasis)
	reg.Register(ast.KindText, t.renderText)
}

func (t *terminalMarkdownRenderer) renderEmphasis(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
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

func (t *terminalMarkdownRenderer) renderText(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if !entering {
		return ast.WalkContinue, nil
	}
	n := node.(*ast.Text)
	w.Write(n.Segment.Value(source))
	return ast.WalkContinue, nil
}

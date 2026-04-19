package terminal

import (
	"bufio"
	"bytes"

	"github.com/joshtyf/inky/core"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/util"
)

type MarkdownRenderer struct {
}

func NewMarkdownRenderer() *MarkdownRenderer {
	return &MarkdownRenderer{}
}

func (m *MarkdownRenderer) Render(line int, es *core.EditorState) (string, error) {
	gm := goldmark.New(
		goldmark.WithRenderer(
			renderer.NewRenderer(
				renderer.WithNodeRenderers(
					util.Prioritized(m, 100),
				),
			),
		),
	)
	// TODO; optimise by caching rendered lines and only re-rendering changed lines
	var buf bytes.Buffer
	err := gm.Convert(es.GetAll(), &buf)
	if err != nil {
		return "", err
	}
	scanner := bufio.NewScanner(&buf)
	for range line + 1 { // Scan until the desired line
		scanner.Scan()
	}
	return scanner.Text(), nil
}

func (m *MarkdownRenderer) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	reg.Register(ast.KindParagraph, m.renderParagraph)
	reg.Register(ast.KindEmphasis, m.renderEmphasis)
	reg.Register(ast.KindText, m.renderText)
}

func (m *MarkdownRenderer) renderParagraph(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if !entering {
		_, err := w.WriteString("\n\n")
		if err != nil {
			panic(err)
		}
	}
	return ast.WalkContinue, nil
}

func (m *MarkdownRenderer) renderEmphasis(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
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
}

func (m *MarkdownRenderer) renderText(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if !entering {
		return ast.WalkContinue, nil
	}
	n := node.(*ast.Text)
	_, err := w.Write(n.Segment.Value(source))
	return ast.WalkContinue, err
}

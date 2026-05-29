package terminal

import (
	"bufio"
	"bytes"
	"fmt"
	"strings"

	"github.com/joshtyf/inky/core"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/util"
)

const (
	codeBlockInnerWidth = 40
	thematicBreakWidth  = 40
)

type MarkdownRenderer struct {
	lastRenderedVersion int
	renderCache         []string
}

func NewMarkdownRenderer() *MarkdownRenderer {
	return &MarkdownRenderer{
		lastRenderedVersion: -1,
		renderCache:         nil,
	}
}

func (m *MarkdownRenderer) LastLine(es core.EditorState) int {
	return len(m.renderCache) - 1
}

func (m *MarkdownRenderer) Render(line int, es core.EditorState) (string, error) {
	if m.renderCache == nil || es.Version != m.lastRenderedVersion {
		gm := goldmark.New(
			goldmark.WithRenderer(
				renderer.NewRenderer(
					renderer.WithNodeRenderers(
						util.Prioritized(m, 100),
					),
				),
			),
		)
		var buf bytes.Buffer
		err := gm.Convert(es.GetAll(), &buf)
		if err != nil {
			return "", err
		}
		scanner := bufio.NewScanner(&buf)
		m.renderCache = make([]string, 0)
		for scanner.Scan() {
			m.renderCache = append(m.renderCache, scanner.Text())
		}
		m.lastRenderedVersion = es.Version
	}
	if line < len(m.renderCache) {
		return m.renderCache[line], nil
	}
	return "", nil
}

func (m *MarkdownRenderer) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	reg.Register(ast.KindDocument, m.renderDocument)
	reg.Register(ast.KindHeading, m.renderHeading)
	reg.Register(ast.KindBlockquote, m.renderBlockquote)
	reg.Register(ast.KindCodeBlock, m.renderCodeBlock)
	reg.Register(ast.KindFencedCodeBlock, m.renderFencedCodeBlock)
	reg.Register(ast.KindHTMLBlock, m.renderHTMLBlock)
	reg.Register(ast.KindList, m.renderList)
	reg.Register(ast.KindListItem, m.renderListItem)
	reg.Register(ast.KindParagraph, m.renderParagraph)
	reg.Register(ast.KindTextBlock, m.renderTextBlock)
	reg.Register(ast.KindThematicBreak, m.renderThematicBreak)
	reg.Register(ast.KindAutoLink, m.renderAutoLink)
	reg.Register(ast.KindCodeSpan, m.renderCodeSpan)
	reg.Register(ast.KindEmphasis, m.renderEmphasis)
	reg.Register(ast.KindImage, m.renderImage)
	reg.Register(ast.KindLink, m.renderLink)
	reg.Register(ast.KindRawHTML, m.renderRawHTML)
	reg.Register(ast.KindText, m.renderText)
	reg.Register(ast.KindString, m.renderString)
}

func (m *MarkdownRenderer) renderDocument(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	return ast.WalkContinue, nil
}

func (m *MarkdownRenderer) renderHeading(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	n := node.(*ast.Heading)
	if entering {
		if _, err := w.WriteString(AnsiBold); err != nil {
			panic(err)
		}
		if _, err := w.WriteString(strings.Repeat("#", n.Level) + " "); err != nil {
			panic(err)
		}
	} else {
		if _, err := w.WriteString(AnsiReset + "\n\n"); err != nil {
			panic(err)
		}
	}
	return ast.WalkContinue, nil
}

func (m *MarkdownRenderer) renderBlockquote(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	return ast.WalkContinue, nil
}

func (m *MarkdownRenderer) renderParagraph(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	n := node.(*ast.Paragraph)
	isBlockquote := n.Parent() != nil && n.Parent().Kind() == ast.KindBlockquote
	if entering {
		if isBlockquote {
			if _, err := w.WriteString(AnsiItalic + "> "); err != nil {
				panic(err)
			}
		}
	} else {
		if isBlockquote {
			if _, err := w.WriteString(AnsiReset + "\n\n"); err != nil { // reset then newlines
				panic(err)
			}
		} else {
			if _, err := w.WriteString("\n\n"); err != nil {
				panic(err)
			}
		}
	}
	return ast.WalkContinue, nil
}

func (m *MarkdownRenderer) renderEmphasis(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	n := node.(*ast.Emphasis)
	if entering {
		tag := AnsiItalic
		if n.Level == 2 {
			tag = AnsiBold
		}
		if _, err := w.WriteString(tag); err != nil {
			panic(err)
		}
	} else {
		if _, err := w.WriteString(AnsiReset); err != nil {
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
	if _, err := w.Write(n.Segment.Value(source)); err != nil {
		panic(err)
	}
	if n.SoftLineBreak() {
		if _, err := w.WriteString(" "); err != nil {
			panic(err)
		}
	}
	if n.HardLineBreak() {
		if _, err := w.WriteString("\n"); err != nil {
			panic(err)
		}
	}
	return ast.WalkContinue, nil
}

func (m *MarkdownRenderer) renderCodeBlock(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if !entering {
		return ast.WalkContinue, nil
	}
	writeCodeBox(w, source, node, "")
	return ast.WalkSkipChildren, nil
}

func (m *MarkdownRenderer) renderFencedCodeBlock(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if !entering {
		return ast.WalkContinue, nil
	}
	n := node.(*ast.FencedCodeBlock)
	lang := ""
	if l := n.Language(source); l != nil {
		lang = string(l)
	}
	writeCodeBox(w, source, node, lang)
	return ast.WalkSkipChildren, nil
}

// writeCodeBox renders a code block as a full box:
//
//	┌── lang ──────────────┐
//	│ code line            │
//	└──────────────────────┘
func writeCodeBox(w util.BufWriter, source []byte, node ast.Node, lang string) {
	const inner = codeBlockInnerWidth
	borderWidth := inner + 2
	if lang != "" {
		label := " " + lang + " "
		dashes := max(borderWidth-2-len(label), 0)
		half := dashes / 2
		if _, err := w.WriteString("┌" + strings.Repeat("─", half) + label + strings.Repeat("─", dashes-half) + "┐\n"); err != nil {
			panic(err)
		}
	} else {
		if _, err := w.WriteString("┌" + strings.Repeat("─", borderWidth-2) + "┐\n"); err != nil {
			panic(err)
		}
	}
	for i := 0; i < node.Lines().Len(); i++ {
		line := node.Lines().At(i)
		content := strings.TrimRight(string(line.Value(source)), "\n")
		padding := inner - 1 - len(content)
		if padding < 0 {
			padding = 0
		}
		if _, err := w.WriteString("│ " + content + strings.Repeat(" ", padding) + "│\n"); err != nil {
			panic(err)
		}
	}
	if _, err := w.WriteString("└" + strings.Repeat("─", borderWidth-2) + "┘\n\n"); err != nil {
		panic(err)
	}
}

func (m *MarkdownRenderer) renderCodeSpan(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if !entering {
		return ast.WalkContinue, nil
	}
	if _, err := w.WriteString(AnsiDim + "`"); err != nil {
		panic(err)
	}
	for c := node.FirstChild(); c != nil; c = c.NextSibling() {
		segment := c.(*ast.Text).Segment
		value := segment.Value(source)
		if bytes.HasSuffix(value, []byte("\n")) {
			if _, err := w.Write(value[:len(value)-1]); err != nil {
				panic(err)
			}
			if _, err := w.WriteString(" "); err != nil {
				panic(err)
			}
		} else {
			if _, err := w.Write(value); err != nil {
				panic(err)
			}
		}
	}
	if _, err := w.WriteString("`" + AnsiReset); err != nil {
		panic(err)
	}
	return ast.WalkSkipChildren, nil
}

func (m *MarkdownRenderer) renderList(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	// Only emit a trailing newline for top-level lists; nested lists (whose
	// parent is a ListItem) must not, or they produce a spurious blank line
	// before the next sibling item in tight lists.
	if !entering && (node.Parent() == nil || node.Parent().Kind() != ast.KindListItem) {
		if _, err := w.WriteString("\n"); err != nil {
			panic(err)
		}
	}
	return ast.WalkContinue, nil
}

func (m *MarkdownRenderer) renderListItem(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if !entering {
		return ast.WalkContinue, nil
	}
	depth := 0
	for p := node.Parent(); p != nil; p = p.Parent() {
		if p.Kind() == ast.KindList {
			depth++
		}
	}
	if depth > 0 {
		depth--
	}
	indent := strings.Repeat("  ", depth)
	parent, ok := node.Parent().(*ast.List)
	if !ok {
		panic("renderListItem: parent is not an *ast.List")
	}
	if parent.IsOrdered() {
		pos := parent.Start
		for sib := node.PreviousSibling(); sib != nil; sib = sib.PreviousSibling() {
			pos++
		}
		if _, err := fmt.Fprintf(w, "%s%d. ", indent, pos); err != nil {
			panic(err)
		}
	} else {
		if _, err := w.WriteString(indent + "• "); err != nil {
			panic(err)
		}
	}
	return ast.WalkContinue, nil
}

func (m *MarkdownRenderer) renderTextBlock(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if !entering {
		if _, err := w.WriteString("\n"); err != nil {
			panic(err)
		}
	}
	return ast.WalkContinue, nil
}

func (m *MarkdownRenderer) renderThematicBreak(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if entering {
		if _, err := w.WriteString(strings.Repeat("─", thematicBreakWidth) + "\n\n"); err != nil {
			panic(err)
		}
	}
	return ast.WalkContinue, nil
}

func (m *MarkdownRenderer) renderLink(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	n := node.(*ast.Link)
	if entering {
		if _, err := w.WriteString("\x1b]8;;" + string(n.Destination) + "\x1b\\"); err != nil {
			panic(err)
		}
	} else {
		if _, err := w.WriteString("\x1b]8;;\x1b\\"); err != nil {
			panic(err)
		}
	}
	return ast.WalkContinue, nil
}

func (m *MarkdownRenderer) renderAutoLink(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if !entering {
		return ast.WalkContinue, nil
	}
	n := node.(*ast.AutoLink)
	url := string(n.URL(source))
	label := string(n.Label(source))
	if _, err := w.WriteString(AnsiHyperlink(url, label)); err != nil {
		panic(err)
	}
	return ast.WalkSkipChildren, nil
}

func (m *MarkdownRenderer) renderImage(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if !entering {
		return ast.WalkContinue, nil
	}
	if _, err := w.WriteString("[image: "); err != nil {
		panic(err)
	}
	for c := node.FirstChild(); c != nil; c = c.NextSibling() {
		if t, ok := c.(*ast.Text); ok {
			if _, err := w.Write(t.Segment.Value(source)); err != nil {
				panic(err)
			}
		}
	}
	if _, err := w.WriteString("]"); err != nil {
		panic(err)
	}
	return ast.WalkSkipChildren, nil
}

func (m *MarkdownRenderer) renderHTMLBlock(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if !entering {
		return ast.WalkContinue, nil
	}
	l := node.Lines().Len()
	for i := range l {
		line := node.Lines().At(i)
		if _, err := w.Write(line.Value(source)); err != nil {
			panic(err)
		}
	}
	return ast.WalkContinue, nil
}

func (m *MarkdownRenderer) renderRawHTML(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if !entering {
		return ast.WalkSkipChildren, nil
	}
	n := node.(*ast.RawHTML)
	for i := 0; i < n.Segments.Len(); i++ {
		seg := n.Segments.At(i)
		if _, err := w.Write(seg.Value(source)); err != nil {
			panic(err)
		}
	}
	return ast.WalkSkipChildren, nil
}

func (m *MarkdownRenderer) renderString(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if !entering {
		return ast.WalkContinue, nil
	}
	n := node.(*ast.String)
	if _, err := w.Write(n.Value); err != nil {
		panic(err)
	}
	return ast.WalkContinue, nil
}

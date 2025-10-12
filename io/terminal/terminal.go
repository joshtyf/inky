package terminal

import (
	"fmt"
	"log"
	"os"
	"strings"
	"unicode/utf8"

	"github.com/joshtyf/texteditor/core"
	editorLog "github.com/joshtyf/texteditor/log"
	"github.com/spf13/viper"
	"golang.org/x/term"
)

func setDefaultConfig(conf *viper.Viper) {
	if conf == nil {
		conf = viper.New()
	}
	conf.SetDefault("showCharCount", true)
	// TODO: create key mapping for terminal input
}

type Selection struct {
	anchorLine       int
	anchorColumn     int
	focusLine        int
	focusColumn      int
	forwardDirection bool
}

func (s *Selection) active() bool {
	return !(s.anchorLine == s.focusLine && s.anchorColumn == s.focusColumn)
}

func (s *Selection) insertHighlightStart(line, col int) bool {
	var cmpLine, cmpCol int
	if s.forwardDirection {
		cmpLine = s.anchorLine
		cmpCol = s.anchorColumn
	} else {
		cmpLine = s.focusLine
		cmpCol = s.focusColumn
	}
	return cmpLine == line && cmpCol == col
}

func (s *Selection) insertHighlightEnd(line, col int) bool {
	var cmpLine, cmpCol int
	if s.forwardDirection {
		cmpLine = s.focusLine
		cmpCol = s.focusColumn
	} else {
		cmpLine = s.anchorLine
		cmpCol = s.anchorColumn
	}
	return cmpLine == line && cmpCol == col
}

type EditorIO struct {
	top           int
	logger        *log.Logger
	input         *input
	output        *output
	showCharCount bool
	selection     *Selection
}

func NewEditorIO(globalConf *viper.Viper) *EditorIO {
	logger := editorLog.CreateLogger("terminalIO")
	conf := globalConf.Sub("terminal")
	setDefaultConfig(conf)
	return &EditorIO{
		top:           0,
		logger:        logger,
		input:         newInput(logger, nil),
		output:        newOutput(logger),
		showCharCount: conf.GetBool("showCharCount"),
		selection: &Selection{
			anchorLine:       0,
			anchorColumn:     0,
			focusLine:        0,
			focusColumn:      0,
			forwardDirection: true,
		},
	}
}

func (io *EditorIO) Start() (<-chan *core.Key, error) {
	err := io.setup()
	if err != nil {
		return nil, fmt.Errorf("error initialising editor io: %w", err)
	}
	inputCh := make(chan *core.Key)
	go func() {
		defer close(inputCh)

		for {
			keys, err := io.input.read()
			if err != nil {
				panic(fmt.Sprintf("error reading input: %v", err))
			}
			for _, k := range keys {
				inputCh <- k
			}
		}
	}()
	return inputCh, nil
}

func (io *EditorIO) DisplayEditor(es *core.EditorState) error {
	_, h, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil {
		return fmt.Errorf("error getting terminal size for display: %w", err)
	}
	// Reposition screen to match editor view
	if es.CurrentLine < io.top {
		io.top = es.CurrentLine
	} else if es.CurrentLine >= io.top+h-1 {
		io.top = es.CurrentLine - h + 2
	}
	// Clear screen
	// TODO: refactor this
	for i := range h {
		fmt.Print(getGoToLineEscapeSequence(i))
		fmt.Print(eraseEntireLine)
	}
	lines, err := es.ReadEditorLines(io.top, h-1) // Last line reserved for status line
	if err != nil {
		return fmt.Errorf("error reading editor lines: %w", err)
	}
	fullContent := make([]byte, 0)
	// Selection
	// TODO: refactor selection. It is working but the code should be neater
	if es.KeyPressed != nil &&
		es.KeyPressed.Code != core.ShiftArrowUp &&
		es.KeyPressed.Code != core.ShiftArrowDown &&
		es.KeyPressed.Code != core.ShiftArrowLeft && es.KeyPressed.Code != core.ShiftArrowRight {
		io.selection.anchorLine = es.CurrentLine - io.top
		io.selection.anchorColumn = es.CurrentColumn
	}
	io.selection.focusLine = es.CurrentLine - io.top
	io.selection.focusColumn = es.CurrentColumn
	if io.selection.focusLine < io.selection.anchorLine || io.selection.focusColumn < io.selection.anchorColumn {
		io.selection.forwardDirection = false
	} else {
		io.selection.forwardDirection = true
	}

	io.logger.Println(io.selection)
	for i := range lines {
		lineColumn := 0
		for j := 0; j < len(lines[i]); j++ {
			if io.selection.active() && io.selection.insertHighlightStart(i, lineColumn) {
				fullContent = append(fullContent, []byte(highlightStart)...)
			}
			r, size := utf8.DecodeRune(lines[i][j:])
			if r == utf8.RuneError {
				if size == 1 {
					return fmt.Errorf("error decoding rune: invalid byte sequence %v", lines[i][j:])
				} else {
					return fmt.Errorf("error decoding rune: empty byte sequence")
				}
			}
			fullContent = append(fullContent, lines[i][j:j+size]...)
			lineColumn++
			j += size - 1
			if io.selection.active() && io.selection.insertHighlightEnd(i, lineColumn) {
				fullContent = append(fullContent, []byte(highlightEnd)...)
			}
		}
		fullContent = append(fullContent, "\n"...)
	}
	io.output.writeData(fullContent)
	io.writeStatusLine(h-1, es)
	io.output.moveCursor(es.CurrentLine-io.top, es.CurrentColumn)
	return nil
}

func (io *EditorIO) writeStatusLine(line int, es *core.EditorState) {
	statusConf := []string{}
	if io.showCharCount {
		statusConf = append(statusConf, fmt.Sprintf("Char Count %d", es.CharCount))
	}
	statusConf = append(statusConf, fmt.Sprintf("Line %d, Column %d", es.CurrentLine+1, es.CurrentColumn+1))
	statusConf = append(statusConf, fmt.Sprintf("Saved: %t", es.EditorSaved))
	status := strings.Join(statusConf, " | ")

	io.output.writeRawLine(line, status)
}

func (io *EditorIO) setup() error {
	if err := io.input.setup(); err != nil {
		return fmt.Errorf("error setting up terminal input: %w", err)
	}
	if err := io.output.setup(); err != nil {
		return fmt.Errorf("error setting up terminal output: %w", err)
	}
	return nil
}

func (io *EditorIO) Close() error {
	if err := io.input.reset(); err != nil {
		return fmt.Errorf("error resetting terminal input: %w", err)
	}
	if err := io.output.reset(); err != nil {
		return fmt.Errorf("error resetting terminal output: %w", err)
	}
	return nil
}

package terminal

import (
	"fmt"
	"log"
	"os"

	"github.com/joshtyf/texteditor/core"
	editorLog "github.com/joshtyf/texteditor/log"
	"golang.org/x/term"
)

type EditorIO struct {
	top    int
	logger *log.Logger
	input  *input
	output *output
}

func NewEditorIO() *EditorIO {
	logger := editorLog.CreateLogger("terminalIO")
	return &EditorIO{
		top:    0,
		logger: logger,
		input:  newInput(logger, nil),
		output: newOutput(logger),
	}
}

func (io *EditorIO) Start(es *core.EditorState) (<-chan *core.Key, error) {
	io.setup()
	err := io.DisplayEditor(es)
	if err != nil {
		return nil, fmt.Errorf("error initialising editor io: %w", err)
	}
	inputCh := make(chan *core.Key)
	go func() {
		defer close(inputCh)

		for {
			k, err := io.input.read()
			if err != nil {
				panic(fmt.Sprintf("error reading input: %v", err))
			}
			inputCh <- k
		}
	}()
	return inputCh, nil
}

func (io *EditorIO) DisplayEditor(es *core.EditorState) error {
	_, h, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil {
		return fmt.Errorf("error getting terminal size for display: %w", err)
	}
	io.output.clearScreen()
	// Reposition screen to match editor view
	if es.CurrentLine < io.top {
		io.top = es.CurrentLine
	} else if es.CurrentLine >= io.top+h-1 {
		io.top = es.CurrentLine - h + 2
	}
	content, err := es.ReadEditorLines(io.top, h-1) // Last line reserved for status line
	if err != nil {
		return fmt.Errorf("error reading editor lines: %w", err)
	}
	for i := range content {
		io.output.writeLine(i, content[i])
	}

	io.output.moveCursor(es.CurrentLine-io.top, es.CurrentColumn)
	return nil
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

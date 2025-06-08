package output

import (
	"fmt"
	"log"
	"os"

	"github.com/joshtyf/texteditor/core"
	editorLog "github.com/joshtyf/texteditor/log"
	"golang.org/x/term"
)

type OutputTerminal struct {
	top                int
	editorStateUpdates chan *core.EditorState
	logger             *log.Logger
}

func NewOutputTerminal() *OutputTerminal {
	ot := &OutputTerminal{
		top:                0,
		editorStateUpdates: make(chan *core.EditorState),
		logger:             editorLog.CreateLogger("output"),
	}

	return ot
}

func (ot *OutputTerminal) StartAndListen(e *core.Editor) error {
	// Initialize the output terminal screen
	if err := ot.init(); err != nil {
		return fmt.Errorf("error initializing output terminal: %w", err)
	}

	// Start listening for editor state updates
	go ot.listen(e)

	return nil
}

func (ot *OutputTerminal) listen(e *core.Editor) {
	e.RegisterListener(ot.editorStateUpdates)

	for editorState := range ot.editorStateUpdates {
		_, h, err := term.GetSize(int(os.Stdout.Fd()))
		if err != nil {
			ot.logger.Printf("error getting terminal size: %v", err)
		}
		// Status line
		if editorState.Error != nil {
			// Last line reserved for status line
			// TODO: refactor updating of status line into a separate function
			ot.writeLine(h-1, fmt.Sprintf("~error~ %s", editorState.Error.Error()))
			continue
		}
		fmt.Printf("%s%s", cursorHome, clearScreen)
		// Reposition screen to match editor view
		if editorState.CurrentLine < ot.top {
			ot.top = editorState.CurrentLine
		} else if editorState.CurrentLine >= ot.top+h-1 {
			ot.top = editorState.CurrentLine - h + 2
		}
		content, err := editorState.ReadEditorLines(ot.top, h-1) // Last line reserved for status line
		if err != nil {
			// TODO: update the status line? how to handle this?
			ot.logger.Printf("error reading lines: %v", err)
		}
		for i := range content {
			ot.writeLine(i, fmt.Sprintf("~ %s", content[i]))
		}
		ot.writeLine(h-1, "~end~")

		// // Reposition cursor to current line and column
		fmt.Printf("\033[%d;%dH", editorState.CurrentLine-ot.top+1, editorState.CurrentColumn+3)
	}
	fmt.Printf("%s%s", cursorHome, clearScreen)
	ot.logger.Println("Closing output terminal")
}

func (ot *OutputTerminal) init() error {
	_, h, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil {
		return err
	}
	fmt.Printf("%s%s", cursorHome, clearScreen)
	for i := range h {
		if err := ot.writeLine(i, "~ "); err != nil {
			return err
		}
	}
	ot.writeLine(h-1, "~end~")
	fmt.Print("\033[1;3H")
	return nil
}

func (ot *OutputTerminal) writeLine(line int, content string) error {
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

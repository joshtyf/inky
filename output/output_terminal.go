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

func (ot *OutputTerminal) Listen(e *core.Editor) {
	e.RegisterListener(ot.editorStateUpdates)
	// Hide cursor
	_, err := os.Stdout.WriteString("\033[?25l")
	if err != nil {
		ot.logger.Printf("error hiding cursor: %v", err)
	}
	defer func() {
		// Show cursor
		_, err = os.Stdout.WriteString("\033[?25h")
		if err != nil {
			ot.logger.Printf("error showing cursor: %v", err)
		}
	}()

	for editorState := range ot.editorStateUpdates {
		fmt.Printf("%s%s", cursorHome, clearScreen)
		_, h, err := term.GetSize(int(os.Stdout.Fd()))
		if err != nil {
			ot.logger.Printf("error getting terminal size: %v", err)
		}
		if editorState.CurrentLine < ot.top {
			ot.top = editorState.CurrentLine
		} else if editorState.CurrentLine >= ot.top+h-1 {
			ot.top = editorState.CurrentLine - h + 2
		}
		content, err := editorState.ReadEditorLines(ot.top, h-1) // Last line reserved for status line
		if err != nil {
			ot.logger.Printf("error reading lines: %v", err)
		}
		for i := range content {
			// TODO: beautify this code
			// Add highlight to the current column current line
			if i == editorState.CurrentLine-ot.top {
				fmt.Printf("[%s", content[i][:editorState.CurrentColumn])
				if editorState.CurrentColumn < len(content[i]) {
					// Highlight the current character
					fmt.Printf("%s%s%s", highlightStart, string(content[i][editorState.CurrentColumn]), highlightEnd)
				} else {
					fmt.Printf("%s %s", highlightStart, highlightEnd)
				}
				if editorState.CurrentColumn+1 < len(content[i]) {
					fmt.Printf("%s", content[i][editorState.CurrentColumn+1:])
				}
				fmt.Println("]")
				continue
			}
			fmt.Printf("[%s]\n", content[i])
		}
		// Status line
		if editorState.Error != nil {
			// Should this check be done at the start?
			fmt.Printf("~error~ %s", editorState.Error.Error())
		} else {
			fmt.Print("~end~")
		}
	}
	fmt.Printf("%s%s", cursorHome, clearScreen)
	ot.logger.Println("Closing output terminal")
}

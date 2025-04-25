package output

import (
	"fmt"
	"log"
	"os"

	"github.com/joshtyf/texteditor/core"
	"golang.org/x/term"
)

type OutputTerminal struct {
	top                int
	editorStateUpdates chan *core.EditorState
}

func NewOutputTerminal() *OutputTerminal {
	return &OutputTerminal{
		top:                0,
		editorStateUpdates: make(chan *core.EditorState),
	}
}

func (ot *OutputTerminal) Listen(e *core.Editor) {
	e.RegisterListener(ot.editorStateUpdates)
	// Hide cursor
	_, err := os.Stdout.WriteString("\033[?25l")
	if err != nil {
		log.Fatalf("error hiding cursor: %v", err)
	}
	defer func() {
		// Show cursor
		_, err = os.Stdout.WriteString("\033[?25h")
		if err != nil {
			log.Printf("error showing cursor: %v", err)
		}
	}()

	for editorState := range ot.editorStateUpdates {
		log.Printf("%s%s", cursorHome, clearScreen)
		_, h, err := term.GetSize(int(os.Stdout.Fd()))
		if err != nil {
			log.Fatalf("error getting terminal size: %v", err)
		}
		if editorState.CurrentLine < ot.top {
			ot.top = editorState.CurrentLine
		} else if editorState.CurrentLine >= ot.top+h-1 {
			ot.top = editorState.CurrentLine - h + 2
		}
		content, err := editorState.ReadEditorLines(ot.top, h-1) // Last line reserved for status line
		if err != nil {
			log.Fatalf("error reading lines: %v", err)
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
				log.Println("]")
				continue
			}
			log.Printf("[%s]", content[i])
		}
		// Status line
		fmt.Print("~end~")
	}
	log.Printf("%s%s", cursorHome, clearScreen)
	log.Println("Closing output terminal")
}

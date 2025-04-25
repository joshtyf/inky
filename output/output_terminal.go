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
		// log.Print(cm.GetInfo())
		// log.Print(buf.GetInfo())
		_, h, err := term.GetSize(int(os.Stdout.Fd()))
		if err != nil {
			log.Fatalf("error getting terminal size: %v", err)
		}
		content, err := editorState.ReadEditorLines(ot.top, h-1)
		if err != nil {
			log.Fatalf("error reading lines: %v", err)
		}
		for i := range content {
			log.Printf("[%s]", content[i])
		}
		// Status line
		fmt.Print("~end~")
	}
	log.Println("Closing output terminal")
}

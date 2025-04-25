package main

import (
	"context"
	"log"
	"os/signal"
	"syscall"
	"time"

	"github.com/joshtyf/texteditor/core"
	"github.com/joshtyf/texteditor/output"
)

func main() {
	// Remove all log prefix
	log.SetFlags(0)

	editor := core.NewEditor(core.NewGapBuffer())
	ot := output.NewOutputTerminal()
	go ot.Listen(editor)
	editorCtx, _ := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	if err := editor.Start(editorCtx, core.NewTerminalInput()); err != nil {
		log.Fatalf("error starting editor: %v", err)
	}
	// TODO: implement wait for proper shutdown
	time.Sleep(time.Second * 5)
}

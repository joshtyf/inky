package main

import (
	"context"
	"errors"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/joshtyf/texteditor/core"
	editorLog "github.com/joshtyf/texteditor/log"
	"github.com/joshtyf/texteditor/output"
)

func main() {
	logFile, err := os.OpenFile("editor.log", os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0666)
	if err != nil {
		log.Printf("error opening log file, %v", err)
	} else {
		editorLog.SetDefaultOutput(logFile)
	}

	editor := core.NewEditor(core.NewGapBuffer())
	ot := output.NewOutputTerminal()
	ot.StartAndListen(editor)
	editorCtx, _ := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	if err := editor.Start(editorCtx, core.NewTerminalInput()); err != nil && errors.Is(err, &core.ErrEditorQuit{}) {
		log.Fatalf("error starting editor: %v", err)
	}
}

package main

import (
	"context"
	"errors"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/joshtyf/texteditor/core"
	io "github.com/joshtyf/texteditor/io/terminal"
	editorLog "github.com/joshtyf/texteditor/log"
)

func main() {
	logFile, err := os.OpenFile("editor.log", os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0666)
	if err != nil {
		log.Printf("error opening log file, %v", err)
	} else {
		editorLog.SetDefaultOutput(logFile)
	}

	editor := core.NewEditor(io.NewEditorIO(), core.NewGapBuffer())
	editorCtx, _ := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	if err := editor.Start(editorCtx); err != nil && errors.Is(err, &core.ErrEditorQuit{}) {
		log.Fatalln(err)
	}
}

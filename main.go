package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/joshtyf/texteditor/core"
	editorIO "github.com/joshtyf/texteditor/io/terminal"
	editorLog "github.com/joshtyf/texteditor/log"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: texteditor <file>")
		os.Exit(1)
	}
	file := os.Args[1]

	logFile, err := os.OpenFile("editor.log", os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0666)
	if err != nil {
		log.Printf("error opening log file, %v", err)
	} else {
		editorLog.SetDefaultOutput(logFile)
	}

	buffer := core.NewGapBuffer()
	editor := core.NewEditor(editorIO.NewEditorIO(), buffer, file)
	editorCtx, _ := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	if err := editor.Start(editorCtx); err != nil && errors.Is(err, &core.ErrEditorQuit{}) {
		log.Fatalln(err)
	}
}

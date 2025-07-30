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
	terminalIO "github.com/joshtyf/texteditor/io/terminal"
	editorLog "github.com/joshtyf/texteditor/log"
	"github.com/spf13/viper"
)

func initConfig() *viper.Viper {
	v := viper.New()
	v.SetConfigName("editor_config")
	v.SetConfigType("yaml")
	// TOOD: update config path
	v.AddConfigPath(".")
	if err := v.ReadInConfig(); err != nil {
		panic(fmt.Errorf("error reading config file: %w", err))
	}
	return v
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: texteditor <file>")
		os.Exit(1)
	}
	file := os.Args[1]
	config := initConfig()
	logFile, err := os.OpenFile("editor.log", os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0666)
	if err != nil {
		log.Printf("error opening log file, %v", err)
	} else {
		editorLog.SetDefaultOutput(logFile)
	}
	// TODO: check that the sub config exists
	editorIO := terminalIO.NewEditorIO(config)
	buffer := core.NewGapBuffer()
	editor := core.NewEditor(editorIO, buffer, file)
	editorCtx, _ := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	if err := editor.Start(editorCtx); err != nil && errors.Is(err, &core.ErrEditorQuit{}) {
		log.Fatalln(err)
	}
}

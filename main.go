package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/joshtyf/inky/buffer"
	"github.com/joshtyf/inky/core"
	"github.com/joshtyf/inky/ui/terminal"
)

func configureLogging() error {
	f, err := os.OpenFile("editor.log", os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0666)
	if err != nil {
		return fmt.Errorf("error opening log file: %w", err)
	}
	// TODO: add levels
	slog.SetDefault(slog.New(slog.NewTextHandler(f, &slog.HandlerOptions{
		AddSource: true,
	})))
	return nil
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: inky <file-path>")
		return
	}
	filePath := os.Args[1]
	// TODO: add flag for log level
	err := configureLogging()
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	editor := core.NewEditor(filePath, terminal.NewTerminal(), buffer.NewGapBuffer())
	ctx, _ := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	if err := editor.Start(ctx); err != nil {
		if _, ok := err.(*core.EditorClosedError); !ok {
			fmt.Println(err.Error())
		}
	}
}

package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/joshtyf/inky/buffer"
	"github.com/joshtyf/inky/core"
	"github.com/joshtyf/inky/ui/terminal"
)

func main() {
	if len(os.Args) < 2 {
		println("Usage: inky <file-path>")
		return
	}
	filePath := os.Args[1]
	editor := core.NewEditor(filePath, terminal.NewTerminal(), buffer.NewGapBuffer())
	ctx, _ := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	if err := editor.Start(ctx); err != nil {
		panic(err)
	}
}

package main

import (
	"context"
	"flag"
	"os/signal"
	"syscall"

	"github.com/joshtyf/inky/buffer"
	"github.com/joshtyf/inky/core"
	"github.com/joshtyf/inky/ui/terminal"
)

func main() {
	var filePath string
	flag.StringVar(&filePath, "file", "", "Path to file to open")
	flag.Parse()
	editor := core.NewEditor(filePath, terminal.NewTerminal(), buffer.NewGapBuffer())
	ctx, _ := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	if err := editor.Start(ctx); err != nil {
		panic(err)
	}
}

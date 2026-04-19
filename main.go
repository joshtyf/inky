package main

import (
	"context"
	"os/signal"
	"syscall"

	"github.com/joshtyf/inky/buffer"
	"github.com/joshtyf/inky/core"
	"github.com/joshtyf/inky/ui/terminal"
)

func main() {
	editor := core.NewEditor(terminal.NewTerminal(), buffer.NewGapBuffer())
	ctx, _ := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	if err := editor.Start(ctx); err != nil {
		panic(err)
	}
}

package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"golang.org/x/term"
)

func main() {
	// Remove all log prefix
	log.SetFlags(0)
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

	// // Read from file
	// fileContent, err := os.ReadFile("sample.txt")
	// if err != nil {
	// 	log.Fatalf("error reading file: %v", err)
	// }
	buf := NewGapBuffer()
	cm := NewCursorMgr(WithBuffer(buf))
	// Set up signal channels
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	// Set up input channels
	input := NewTerminalInput()
	inputCtx := startAndListen(input)

	top := 0

	for {
		select {
		case <-sigChan:
			log.Println("Aborting text editor")
			input.stop()
			return
		case <-inputCtx.Done():
			log.Println("Closing text editor")
			input.stop()
			return
		case k := <-inputCtx.inputCh:
			switch k.Code {
			case RuneKey:
				for _, r := range k.Runes {
					data := []byte(string(r))
					for i := range data {
						cm.InsertAtCursor(data[i], buf)
					}
				}
			case ArrowUp:
				cm.MoveCursorUp()
				// TODO: Update top line
			case ArrowDown:
				cm.MoveCursorDown()
				// TODO: Update top line
			case ArrowLeft:
				cm.MoveCursorLeft()
			case ArrowRight:
				cm.MoveCursorRight()
			case Newline:
				cm.InsertAtCursor('\n', buf)
			case Backspace:
				cm.BackspaceAtCursor(buf)
			case Undo:
				cm.Undo(buf)
			}
		}
		// log.Print(buf.GetInfo())
		log.Printf("%s%s", cursorHome, clearScreen)
		// log.Print(cm.GetInfo())
		// log.Print(buf.GetInfo())
		_, h, err := term.GetSize(int(os.Stdout.Fd()))
		if err != nil {
			log.Fatalf("error getting terminal size: %v", err)
		}
		content, err := cm.ReadLines(top, h-1, buf)
		if err != nil {
			log.Fatalf("error reading lines: %v", err)
		}
		for i := range content {
			log.Printf("[%s]", content[i])
		}
		// Status line
		fmt.Print("~end~")
	}
}

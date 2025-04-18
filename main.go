package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	// Remove all log prefix
	log.SetFlags(0)

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
	inputCtx := startInput(DEFAULT_INPUT)

	for {
		select {
		case <-sigChan:
			log.Println("Aborting text editor")
			return
		case <-inputCtx.Done():
			log.Println("Closing text editor")
			return
		case input := <-inputCtx.charOut:
			cm.InsertAtCursor(input, buf)
		case input := <-inputCtx.keyOut:
			switch input {
			case ArrowUp:
				cm.MoveCursorUp()
			case ArrowDown:
				cm.MoveCursorDown()
			case ArrowLeft:
				cm.MoveCursorLeft()
			case ArrowRight:
				cm.MoveCursorRight()
			case NewLine:
				cm.InsertAtCursor('\n', buf)
			case Delete:
				cm.BackspaceAtCursor(buf)
			case Undo:
				cm.Undo(buf)
			}
		}
		// log.Print(buf.GetInfo())
		log.Printf("%s%s", cursorHome, clearScreen)
		log.Print(cm.GetInfo())
		log.Print(buf.GetInfo())
		log.Print(cm.ReturnLine(buf))
	}
}

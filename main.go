package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"
)

func startInput(i Input, km KeyMap) (<-chan byte, <-chan SpecialKey) {
	mappedCharOut := make(chan byte)
	mappedKeyOut := make(chan SpecialKey)
	transmitCharOut := make(chan byte)
	transmitKeyOut := make(chan SpecialKey)

	go i.transmit(transmitCharOut, transmitKeyOut)
	go func() {
		for {
			select {
			case b := <-transmitCharOut:
				km.mapByte(b, mappedCharOut, mappedKeyOut)
			case k := <-transmitKeyOut:
				km.mapSpecialKey(k, mappedCharOut, mappedKeyOut)
			}
		}
	}()
	return mappedCharOut, mappedKeyOut
}

func main() {
	// Remove all log prefix
	log.SetFlags(0)

	// // Read from file
	fileContent, err := os.ReadFile("sample.txt")
	if err != nil {
		log.Fatalf("error reading file: %v", err)
	}
	buf := NewGapBufferWithContent(fileContent)
	cm := NewCursorMgr(WithBuffer(buf))

	// Set up signal channels
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	// Set up input channels
	charInput, keyInput := startInput(DEFAULT_INPUT, DEFAULT_KEYMAP)

	for {
		select {
		case <-sigChan:
			log.Println("Received signal, exiting")
			return
		case input := <-charInput:
			cm.InsertAtCursor(input, buf)
		case input := <-keyInput:
			// Clear the screen
			log.Printf("%s%s", cursorHome, clearScreen)
			switch input {
			case ArrowLeft:
				cm.MoveCursorLeft()
			case ArrowRight:
				cm.MoveCursorRight()
			case ArrowUp:
				cm.MoveCursorUp()
			case ArrowDown:
				cm.MoveCursorDown()
			case Delete:
				cm.BackspaceAtCursor(buf)
			case NewLine:
				cm.InsertAtCursor('\n', buf)
			}
		}
		// log.Print(buf.GetInfo())
		log.Print(cm.GetInfo())
		log.Print(cm.ReturnLine(buf))
	}
}

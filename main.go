package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"
)

func startInput(i Input) (<-chan byte, <-chan SpecialKey) {
	charOut := make(chan byte)
	keyOut := make(chan SpecialKey)

	go i.transmit(charOut, keyOut)

	return charOut, keyOut
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
	cc := NewCommandController(cm, buf)
	// Set up signal channels
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	// Set up input channels
	charInput, keyInput := startInput(DEFAULT_INPUT)
	// Set up command channel
	commandCh := cc.processInput(charInput, keyInput)

	for {
		select {
		case <-sigChan:
			log.Println("Received signal, exiting")
			return
		case command := <-commandCh:
			log.Printf("%s%s", cursorHome, clearScreen)
			command.execute()
		}
		// log.Print(buf.GetInfo())
		log.Print(cm.GetInfo())
		log.Print(cm.ReturnLine(buf))
	}
}

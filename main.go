package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"golang.org/x/sys/unix"
)

type ArrowKey int

const (
	ArrowUp ArrowKey = iota
	ArrowDown
	ArrowLeft
	ArrowRight
	Delete
)

func readUserInput(charInput chan<- byte, keyInput chan<- ArrowKey) {
	// TODO: Make it cross-platform
	termios, err := unix.IoctlGetTermios(int(os.Stdin.Fd()), unix.TIOCGETA)
	if err != nil {
		log.Fatalf("error getting terminal attributes: %v", err)
		return
	}
	termios.Lflag &^= unix.ICANON | unix.ECHO
	err = unix.IoctlSetTermios(int(os.Stdin.Fd()), unix.TIOCSETA, termios)
	if err != nil {
		log.Fatalf("error setting terminal attributes: %v", err)
		return
	}
	defer func() {
		termios.Lflag |= unix.ICANON | unix.ECHO
		err := unix.IoctlSetTermios(int(os.Stdin.Fd()), unix.TIOCSETA, termios)
		if err != nil {
			log.Fatalf("error resetting terminal attributes: %v", err)
		}
	}()
	// Read from stdin
	for {
		var b [3]byte
		n, err := os.Stdin.Read(b[:])
		if err != nil {
			log.Printf("error reading from stdin: %v", err)
			return
		}
		// Assume that the input is ASCII
		if n == 1 {
			switch b[0] {
			case 127:
				keyInput <- Delete
			default:
				charInput <- b[0]
			}
			continue
		} else if n == 3 && b[0] == 0x1b {
			switch b[2] {
			case 'A':
				keyInput <- ArrowUp
			case 'B':
				keyInput <- ArrowDown
			case 'C':
				keyInput <- ArrowRight
			case 'D':
				keyInput <- ArrowLeft
			}
		} else {
			log.Printf("Received unexpected input: %v", b[:n])
		}
	}
}

func main() {
	// Remove all log prefix
	log.SetFlags(0)

	buf := NewGapBuffer()
	cm := NewCursorMgr(buf)

	// // Read from file
	// fileBuffer := NewFileBuffer("sample.txt")
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	charInput := make(chan byte)
	keyInput := make(chan ArrowKey)
	go readUserInput(charInput, keyInput)
	for {
		select {
		case <-sigChan:
			log.Println("Received signal, exiting")
			return
		case input := <-charInput:
			cm.InsertByte(input)
		case input := <-keyInput:
			// Clear the screen
			log.Printf("%s%s", cursorHome, clearScreen)
			switch input {
			case ArrowLeft:
				cm.MoveCursorLeft(1)
			case ArrowRight:
				cm.MoveCursorRight(1)
			case ArrowUp:
				cm.MoveCursorUp(1)
			case ArrowDown:
				cm.MoveCursorDown(1)
			case Delete:
				cm.BackspaceAtCursor()
			}
		}
		log.Print(buf.GetInfo())
		log.Print(cm.GetInfo())
		log.Print(cm.ReturnLine())
	}
}

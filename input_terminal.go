package main

import (
	"log"
	"os"

	"golang.org/x/sys/unix"
)

type TerminalInput struct {
}

func NewTerminalInput() *TerminalInput {
	return &TerminalInput{}
}

func (ti *TerminalInput) readUserInput(charInput chan<- byte, keyInput chan<- ArrowKey) {
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
			case 10:
				keyInput <- NewLine
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

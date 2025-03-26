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

func (ti *TerminalInput) transmit(charOut chan<- byte, keyOut chan<- SpecialKey) {
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
				keyOut <- NewLine
			case 127:
				keyOut <- Delete
			default:
				charOut <- b[0]
			}
			continue
		} else if n == 3 && b[0] == 0x1b {
			switch b[2] {
			case 'A':
				keyOut <- ArrowUp
			case 'B':
				keyOut <- ArrowDown
			case 'C':
				keyOut <- ArrowRight
			case 'D':
				keyOut <- ArrowLeft
			}
		} else {
			log.Printf("Received unexpected input: %v", b[:n])
		}
	}
}

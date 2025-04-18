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
	// Hide cursor
	_, err = os.Stdout.WriteString("\033[?25l")
	if err != nil {
		log.Printf("error hiding cursor: %v", err)
		return
	}

	defer func() {
		log.Println("Restoring terminal attributes")
		termios.Lflag |= unix.ICANON | unix.ECHO
		err := unix.IoctlSetTermios(int(os.Stdin.Fd()), unix.TIOCSETA, termios)
		if err != nil {
			log.Fatalf("error resetting terminal attributes: %v", err)
		}
		// Show cursor
		_, err = os.Stdout.WriteString("\033[?25h")
		if err != nil {
			log.Printf("error showing cursor: %v", err)
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
		// TODO: Implement a separate key mapper. Refer to https://github.com/atomicgo/keyboard
		if n == 1 {
			switch b[0] {
			case 4: // Ctrl-D
				log.Println("Received stop signal, exiting")
				return
			case 10:
				keyOut <- NewLine
			case 31: // Currently customied for cmd+z in VSCode
				// TODO: read a user config file to get the key mapping
				keyOut <- Undo
			case 127:
				keyOut <- Delete
			default:
				charOut <- b[0]
			}
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
			default:
				log.Printf("Received unexpected input: %v", b[:n])
			}
		} else {
			log.Printf("Received unexpected input: %v", b[:n])
		}
	}
}

package main

import (
	"log"
	"os"
	"unicode/utf8"

	"golang.org/x/sys/unix"
)

const DEFAULT_BUFFER_READ_SIZE = 256

type TerminalInput struct {
	mapping map[string]Key
}

func NewTerminalInput() *TerminalInput {
	// TODO: allow for custom sequences
	return &TerminalInput{
		mapping: defaultMapping,
	}
}

func (ti *TerminalInput) listen(keyOut chan<- Key) {
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
		var b [DEFAULT_BUFFER_READ_SIZE]byte
		n, err := os.Stdin.Read(b[:])
		if err != nil {
			log.Printf("error reading from stdin: %v", err)
			return
		}

		if k, ok := ti.mapping[string(b[:n])]; ok {
			// TODO: remove this once the 'view' settings have been refactored out
			if k.Code == CtrlD {
				log.Println("Received stop signal, exiting")
				return
			}
			keyOut <- k
			continue
		}

		runes := make([]rune, 0)
		for i := 0; i < n; i++ {
			r, width := utf8.DecodeRune(b[i:])
			if r == utf8.RuneError {
				log.Fatalf("error decoding rune: %v", b)
			}
			runes = append(runes, r)
			i += width - 1
		}

		if len(runes) > 0 {
			keyOut <- Key{Code: RuneKey, Runes: runes}
			continue
		}
	}
}

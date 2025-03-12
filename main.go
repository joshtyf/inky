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
		if n == 1 {
			charInput <- b[0]
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

type FileBuffer struct {
	cursor int
	buffer []byte
}

func NewFileBuffer(filepath string) *FileBuffer {
	b, err := os.ReadFile(filepath)
	if err != nil {
		log.Fatalf("error opening file: %v", err)
		return nil
	}
	return &FileBuffer{
		cursor: 0,
		buffer: b,
	}
}

func (fb *FileBuffer) MoveCursor(offset int) {
	fb.cursor += offset
	if fb.cursor < 0 {
		fb.cursor = 0
	}
	if fb.cursor > len(fb.buffer) {
		fb.cursor = len(fb.buffer)
	}
}

func (fb *FileBuffer) Println() {
	var end int
	for i := fb.cursor; i < len(fb.buffer); i++ {
		if fb.buffer[i] == '\n' {
			end = i
			break
		}
	}
	log.Printf("%s", fb.buffer[fb.cursor:end])
}

func main() {
	// Remove all log prefix
	log.SetFlags(0)

	// Read from file
	fileBuffer := NewFileBuffer("sample.txt")
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
			log.Printf("Received input: %c", input)
		case input := <-keyInput:
			// Clear the screen
			log.Printf("%s%s", cursorHome, clearScreen)
			switch input {
			case ArrowLeft:
				fileBuffer.MoveCursor(-1)
			case ArrowRight:
				fileBuffer.MoveCursor(1)
			case ArrowUp:
				log.Println("ArrowUp")
			case ArrowDown:
				log.Println("ArrowDown")
			}
			fileBuffer.Println()
		}
	}
}

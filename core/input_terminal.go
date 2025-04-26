package core

import (
	"context"
	"errors"
	"log"
	"os"
	"unicode/utf8"

	editorLog "github.com/joshtyf/texteditor/log"
	"golang.org/x/sys/unix"
)

const DEFAULT_BUFFER_READ_SIZE = 256

type TerminalInput struct {
	mapping map[string]Key
	logger  *log.Logger
}

func NewTerminalInput() *TerminalInput {
	// TODO: allow for custom sequences
	return &TerminalInput{
		mapping: defaultMapping,
		logger:  editorLog.CreateLogger("input"),
	}
}

func (ti *TerminalInput) start(ctx context.Context, cancelInput context.CancelCauseFunc, inputCh chan<- *Key) error {
	ti.setup()
	defer ti.teardown()

	go func() {
		for {
			k, err := ti.listen()
			if err != nil {
				cancelInput(err)
			}
			inputCh <- k
		}
	}()
	<-ctx.Done()
	return nil
}

func (ti *TerminalInput) setup() error {
	// TODO: Make it cross-platform
	termios, err := unix.IoctlGetTermios(int(os.Stdin.Fd()), unix.TIOCGETA)
	if err != nil {
		ti.logger.Println("error getting terminal attributes")
		return err
	}
	termios.Lflag &^= unix.ICANON | unix.ECHO
	ti.logger.Printf("setting terminal attributes: %v", termios.Lflag)
	err = unix.IoctlSetTermios(int(os.Stdin.Fd()), unix.TIOCSETA, termios)
	if err != nil {
		log.Println("error setting terminal attributes")
		return err
	}
	return nil
}

func (ti *TerminalInput) teardown() error {
	ti.logger.Println("stopping input")
	termios, err := unix.IoctlGetTermios(int(os.Stdin.Fd()), unix.TIOCGETA)
	if err != nil {
		ti.logger.Println("error getting terminal attributes")
	}
	termios.Lflag |= unix.ICANON | unix.ECHO
	err = unix.IoctlSetTermios(int(os.Stdin.Fd()), unix.TIOCSETA, termios)
	if err != nil {
		ti.logger.Printf("error resetting terminal attributes: %v", err)
	}
	return nil
}

func (ti *TerminalInput) listen() (*Key, error) {
	// Read from stdin
	var b [DEFAULT_BUFFER_READ_SIZE]byte
	n, err := os.Stdin.Read(b[:])
	if err != nil {
		ti.logger.Printf("error reading from stdin: %v", err)
		return nil, err
	}

	if k, ok := ti.mapping[string(b[:n])]; ok {
		return &k, nil
	}

	runes := make([]rune, 0)
	for i := 0; i < n; i++ {
		r, width := utf8.DecodeRune(b[i:])
		if r == utf8.RuneError {
			ti.logger.Println("error decoding rune: %v", b)
		}
		runes = append(runes, r)
		i += width - 1
	}

	if len(runes) > 0 {
		return &Key{Code: RuneKey, Runes: runes}, nil
	}
	return nil, errors.New("no key found")
}

package terminal

import (
	"fmt"
	"log"
	"os"
	"unicode/utf8"

	"github.com/joshtyf/texteditor/core"
	"golang.org/x/sys/unix"
)

const default_buffer_read_size = 256

type keyMapping map[string]core.Key

var defaultMapping = keyMapping{
	// Arrow Keys
	"\x1b[A": {Code: core.ArrowUp},
	"\x1b[B": {Code: core.ArrowDown},
	"\x1b[D": {Code: core.ArrowLeft},
	"\x1b[C": {Code: core.ArrowRight},

	// Control Keys
	"\x04": {Code: core.CtrlD},
	"\x7f": {Code: core.Backspace},
	"\x1f": {Code: core.Undo},
	"\x0a": {Code: core.Newline},
	"\x13": {Code: core.Save},
}

type input struct {
	logger  *log.Logger
	mapping keyMapping
}

func newInput(l *log.Logger, m *keyMapping) *input {
	i := &input{
		logger: l,
	}
	if m != nil {
		i.mapping = *m
	} else {
		i.mapping = defaultMapping
	}
	return i
}

func (i *input) setup() error {
	termios, err := unix.IoctlGetTermios(int(os.Stdin.Fd()), unix.TIOCGETA)
	if err != nil {
		return fmt.Errorf("error getting stdin terminal attributes: %w", err)
	}
	termios.Lflag &^= unix.ICANON | unix.ECHO
	termios.Iflag &^= unix.IXON // Disable flow control for Ctrl+S
	err = unix.IoctlSetTermios(int(os.Stdin.Fd()), unix.TIOCSETA, termios)
	if err != nil {
		return fmt.Errorf("error setting stdin terminal attributes: %w", err)
	}
	return nil
}

// TODO: fix bug when stdin reads multiple keys at once
// This can happen when the user holds down a key, causing multiple key events to be read
// Return []*core.Key instead of a single *core.Key
func (i *input) read() (*core.Key, error) {
	// Read from stdin
	var b [default_buffer_read_size]byte
	n, err := os.Stdin.Read(b[:])
	if err != nil {
		return nil, err
	}

	if k, ok := i.mapping[string(b[:n])]; ok {
		return &k, nil
	}

	runes := make([]rune, 0)
	for j := 0; j < n; j++ {
		r, size := utf8.DecodeRune(b[j:])
		if r == utf8.RuneError {
			if size == 1 {
				return nil, fmt.Errorf("error decoding rune: invalid byte sequence %v", b[j:])
			} else {
				return nil, fmt.Errorf("error decoding rune: empty byte sequence")
			}
		}
		runes = append(runes, r)
		j += size - 1
	}

	if len(runes) > 0 {
		return &core.Key{Code: core.RuneKey, Runes: runes}, nil
	} else {
		panic("error reading from stdin: unable to create a key from input")
	}
}

func (i *input) reset() error {
	// TODO: Make it cross-platform
	termios, err := unix.IoctlGetTermios(int(os.Stdin.Fd()), unix.TIOCGETA)
	if err != nil {
		return fmt.Errorf("error getting stdin terminal attributes: %w", err)
	}
	termios.Lflag &^= unix.ICANON | unix.ECHO
	termios.Iflag &^= unix.IXON // Restore flow control for Ctrl+S
	err = unix.IoctlSetTermios(int(os.Stdin.Fd()), unix.TIOCSETA, termios)
	if err != nil {
		return fmt.Errorf("error resetting stdin terminal attributes: %w", err)
	}
	return nil
}

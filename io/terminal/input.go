package terminal

import (
	"fmt"
	"log"
	"os"
	"strings"
	"unicode/utf8"

	"github.com/joshtyf/texteditor/core"
	"golang.org/x/sys/unix"
)

const default_buffer_read_size = 256

type keyMapping map[string]core.Key

var defaultMapping = keyMapping{
	// Escape Sequences
	"\x1b[A":    {Code: core.ArrowUp},
	"\x1b[B":    {Code: core.ArrowDown},
	"\x1b[D":    {Code: core.ArrowLeft},
	"\x1b[C":    {Code: core.ArrowRight},
	"\x1b[1;2A": {Code: core.ShiftArrowUp},
	"\x1b[1;2B": {Code: core.ShiftArrowDown},
	"\x1b[1;2C": {Code: core.ShiftArrowRight},
	"\x1b[1;2D": {Code: core.ShiftArrowLeft},
	"\x1b":      {Code: core.Escape},

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

func (in *input) setup() error {
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

func (in *input) parseSpecialSequences(b []byte) (*core.Key, int) {
	bufstr := string(b)
	var matchedKey *core.Key
	matchedKeySize := 0
	for k, v := range in.mapping {
		predicateLen := len(k)
		if len(bufstr) < predicateLen {
			continue
		}
		// Longest prefix match
		if strings.HasPrefix(bufstr, k) && matchedKeySize < predicateLen {
			matchedKey = &v
			matchedKeySize = predicateLen
		}
	}
	return matchedKey, matchedKeySize
}

func (in *input) read() ([]*core.Key, error) {
	var b [default_buffer_read_size]byte
	n, err := os.Stdin.Read(b[:])
	if err != nil {
		return nil, fmt.Errorf("error reading from stdin: %w", err)
	}
	keys := make([]*core.Key, 0)
	for i := 0; i < n; i++ {
		k, size := in.parseSpecialSequences(b[i:])
		if k != nil {
			keys = append(keys, k)
			i += size - 1 // Move to the next character after the escape sequence
			continue
		}

		// If we reach here, then we try to decode the bytes as a rune
		r, size := utf8.DecodeRune(b[i:])
		if r == utf8.RuneError {
			if size == 1 {
				return nil, fmt.Errorf("error decoding rune: invalid byte sequence %v", b[i:])
			} else {
				return nil, fmt.Errorf("error decoding rune: empty byte sequence")
			}
		}
		keys = append(keys, &core.Key{Code: core.RuneKey, Rune: r})
		i += size - 1 // Move to the next sequence
	}
	return keys, nil
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

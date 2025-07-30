package terminal

import (
	"fmt"
	"log"
	"os"
	"strings"
	"unicode/utf8"

	editorIO "github.com/joshtyf/texteditor/io"
	"golang.org/x/sys/unix"
)

const default_buffer_read_size = 256

type keyMapping map[string]editorIO.Key

var defaultMapping = keyMapping{
	// Escape Sequences
	"\x1b[A": {Code: editorIO.ArrowUp},
	"\x1b[B": {Code: editorIO.ArrowDown},
	"\x1b[D": {Code: editorIO.ArrowLeft},
	"\x1b[C": {Code: editorIO.ArrowRight},
	"\x1b":   {Code: editorIO.Escape},

	// Control Keys
	"\x04": {Code: editorIO.CtrlD},
	"\x7f": {Code: editorIO.Backspace},
	"\x1f": {Code: editorIO.Undo},
	"\x0a": {Code: editorIO.Newline},
	"\x13": {Code: editorIO.Save},
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

func (in *input) parseSpecialSequences(b []byte) (*editorIO.Key, int) {
	bufstr := string(b)
	var matchedKey *editorIO.Key
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

func (in *input) read() ([]*editorIO.Key, error) {
	var b [default_buffer_read_size]byte
	n, err := os.Stdin.Read(b[:])
	if err != nil {
		return nil, fmt.Errorf("error reading from stdin: %w", err)
	}
	keys := make([]*editorIO.Key, 0)
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
		keys = append(keys, &editorIO.Key{Code: editorIO.RuneKey, Rune: r})
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

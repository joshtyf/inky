package main

import (
	"log"
	"os"
	"unicode/utf8"
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

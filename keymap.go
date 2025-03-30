package main

import "log"

func mapBytes(b []byte, n int, charOut chan<- byte, keyOut chan<- SpecialKey) {
	if n == 1 {
		switch b[0] {
		case 10:
			keyOut <- NewLine
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
		}
	} else {
		log.Printf("Received unexpected input: %v", b[:n])
	}
}

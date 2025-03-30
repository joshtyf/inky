package main

import "log"

func mapBytes(b []byte, n int, charOut chan<- byte, keyOut chan<- SpecialKey) {
	if n == 1 {
		switch b[0] {
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

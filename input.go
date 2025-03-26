package main

type SpecialKey int

const (
	ArrowUp SpecialKey = iota
	ArrowDown
	ArrowLeft
	ArrowRight
	Delete
	NewLine
)

var DEFAULT_INPUT = NewTerminalInput()

type Input interface {
	transmit(charOut chan<- byte, keyOut chan<- SpecialKey)
}

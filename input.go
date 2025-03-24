package main

type ArrowKey int

const (
	ArrowUp ArrowKey = iota
	ArrowDown
	ArrowLeft
	ArrowRight
	Delete
	NewLine
)

type InputReader interface {
	readUserInput(charInput chan<- byte, keyInput chan<- ArrowKey)
}

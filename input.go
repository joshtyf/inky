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

type Input interface {
	receive(charInput chan<- byte, keyInput chan<- ArrowKey)
}

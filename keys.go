package main

type KeyCode int

const (
	RuneKey KeyCode = iota
	ArrowUp
	ArrowDown
	ArrowLeft
	ArrowRight
	CtrlD
	Backspace
	Undo
	Newline
)

type Key struct {
	Code  KeyCode
	Runes []rune
}

package core

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
	Save
	Escape
)

type Key struct {
	Code KeyCode
	Rune rune
}

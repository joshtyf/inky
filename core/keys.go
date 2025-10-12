package core

type KeyCode int

const (
	RuneKey KeyCode = iota
	ArrowUp
	ArrowDown
	ArrowLeft
	ArrowRight
	ShiftArrowUp
	ShiftArrowDown
	ShiftArrowLeft
	ShiftArrowRight
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

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

func (k KeyCode) IsShiftArrow() bool {
	return k == ShiftArrowUp || k == ShiftArrowDown || k == ShiftArrowLeft || k == ShiftArrowRight
}

type Key struct {
	Code KeyCode
	Rune rune
}

package core

import (
	"context"
	"fmt"
	"io"
	"unicode/utf8"

	"github.com/joshtyf/inky/log"
)

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
	Save
	Escape
)

type ViewMode int

const (
	MarkdownView ViewMode = iota
	RawView
)

func (k KeyCode) IsMovementKey() bool {
	return k == ArrowUp || k == ArrowDown || k == ArrowLeft || k == ArrowRight
}

func (k KeyCode) IsShiftArrow() bool {
	return k == ShiftArrowUp || k == ShiftArrowDown || k == ShiftArrowLeft || k == ShiftArrowRight
}

func (k KeyCode) IsEditorOperation() bool {
	return k == RuneKey || k == Backspace
}

type Key struct {
	Code KeyCode
	Rune rune
}

type UserInterface interface {
	Init() error
	Close() error
	GetKey(ctx context.Context) <-chan *Key
	Update(es *EditorState) error
}

type Buffer interface {
	InsertRune(r rune, cursor int)
	SeekToChar(cursor int, char byte, count int) int
	ReverseSeekToChar(cursor int, char byte, count int) int
	Read(cursor int, length int) []byte
	ReadAll() []byte
	DeleteRune(cursor int) rune
	GetRune(cursor int) rune
	Len() int
	WriteTo(w io.Writer) (int64, error)
}

type EditorState struct {
	LastKeyPresssed *Key
	CurrentLine     int
	CurrentCol      int
	GetLine         func(lineNumber int) []rune // TODO: should we return bytes or runes?
	GetAll          func() []byte
	ViewMode        ViewMode
}

type Editor struct {
	ui          UserInterface
	buf         Buffer
	lineMap     []int
	currentLine int
	currentCol  int
	viewMode    ViewMode
}

func NewEditor(ui UserInterface, buf Buffer) *Editor {
	lineMap := make([]int, 1)
	return &Editor{
		ui:          ui,
		buf:         buf,
		lineMap:     lineMap,
		currentLine: 0,
		currentCol:  0,
		viewMode:    RawView,
	}
}

func (e *Editor) Start(ctx context.Context) error {
	if err := e.ui.Init(); err != nil {
		return err
	}
	defer e.ui.Close()

	keyCh := e.ui.GetKey(ctx)

	for {
		select {
		case <-ctx.Done():
			return nil
		case key, ok := <-keyCh:
			if !ok {
				return nil
			}
			e.handleKey(key)
			// Step 1: Handle the received key (store in buffer, update cursor, etc.)
			// Step 2: Render the updated state to the ui
			// fmt.Print("Curr line:", e.currentLine, " Col:", e.currentCol, "\n")
			// fmt.Printf("Received key: %+v\n", key)
			e.ui.Update(&EditorState{
				LastKeyPresssed: key,
				CurrentLine:     e.currentLine,
				CurrentCol:      e.currentCol,
				GetLine:         e.getLine,
				GetAll:          e.getAll,
				ViewMode:        e.viewMode,
			})
		}
	}
}

func (e *Editor) handleKey(key *Key) {
	if e.viewMode == MarkdownView {
		switch key.Code {
		case Escape:
			e.toggleViewMode()
		}
	} else {
		switch key.Code {
		case RuneKey:
			e.insertRune(key.Rune)
		case ArrowUp:
			e.moveCursorUp()
		case ArrowDown:
			e.moveCursorDown()
		case ArrowLeft:
			e.moveCursorLeft()
		case ArrowRight:
			e.moveCursorRight()
		case Backspace:
			e.backspace()
		case Escape:
			e.toggleViewMode()
		}
	}
	e.debug()
}

func (e *Editor) moveCursorUp() {
	if e.currentLine == 0 {
		return
	}
	e.currentLine--
	e.currentCol = min(e.currentCol, e.lineMap[e.currentLine]-1) // -1 because we want to be on the last character of the previous line, not the newline char
}

func (e *Editor) moveCursorDown() {
	if e.currentLine >= len(e.lineMap)-1 {
		return
	}
	e.currentLine++
	limit := e.lineMap[e.currentLine]
	if e.currentLine < len(e.lineMap)-1 {
		limit-- // -1 because we want to be on the last character of the line, not the newline char
	}
	e.currentCol = min(e.currentCol, limit)
}

func (e *Editor) moveCursorRight() {
	if e.currentCol < e.lineMap[e.currentLine] {
		e.currentCol++
	}
	// If we're at the end of the line and there is another line,
	// move to the beginning of the next line
	if e.currentCol == e.lineMap[e.currentLine] && e.currentLine < len(e.lineMap)-1 {
		e.currentLine++
		e.currentCol = 0
	}
}

func (e *Editor) moveCursorLeft() {
	// If we're at the beginning of the line and there is a previous line,
	// move to the end (newline char) of the previous line
	if e.currentCol == 0 && e.currentLine > 0 {
		e.currentLine--
		e.currentCol = e.lineMap[e.currentLine]
	}
	if e.currentCol > 0 {
		e.currentCol--
	}
}

func (e *Editor) getCursorPosition() int {
	cursor := 0
	for i := 0; i < e.currentLine; i++ {
		cursor += e.lineMap[i]
	}
	return cursor + e.currentCol
}

func (e *Editor) setCursorPosition(cursor int) {
	if cursor < 0 || cursor > e.buf.Len() {
		panic(fmt.Sprintf("Cursor position %d is out of bounds, total length: %d", cursor, e.buf.Len()))
	}
	line := 0
	for line < len(e.lineMap) && cursor >= e.lineMap[line] {
		cursor -= e.lineMap[line]
		line++
	}
	e.currentLine = line
	e.currentCol = cursor
}

func (e *Editor) insertRune(r rune) Operation {
	cursor := e.getCursorPosition()
	e.buf.InsertRune(r, cursor)
	if r == '\n' {
		newLineMap := make([]int, len(e.lineMap)+1)
		copy(newLineMap, e.lineMap[:e.currentLine+1])
		newLineMap[e.currentLine] = e.currentCol
		newLineMap[e.currentLine+1] = e.lineMap[e.currentLine] - e.currentCol
		copy(newLineMap[e.currentLine+2:], e.lineMap[e.currentLine+1:])
		e.lineMap = newLineMap
	}
	e.lineMap[e.currentLine]++
	e.moveCursorRight()
}

func (e *Editor) backspace() {
	cursor := e.getCursorPosition()
	if cursor == 0 {
		return
	}
	r := e.buf.DeleteRune(cursor - 1)
	e.moveCursorLeft()
	e.lineMap[e.currentLine]--
	if r == '\n' {
		newLineMap := make([]int, len(e.lineMap)-1)
		copy(newLineMap, e.lineMap[:e.currentLine+1])
		newLineMap[e.currentLine] = e.lineMap[e.currentLine] + e.lineMap[e.currentLine+1]
		copy(newLineMap[e.currentLine+1:], e.lineMap[e.currentLine+2:])
		e.lineMap = newLineMap
	}
}

func (e *Editor) toggleViewMode() {
	if e.viewMode == MarkdownView {
		e.viewMode = RawView
	} else {
		e.viewMode = MarkdownView
	}
}

func (e *Editor) getLine(lineNumber int) []rune {
	if lineNumber < 0 || lineNumber >= len(e.lineMap) {
		return []rune{} // TODO: figure out what the right behaviour is here.
	}
	cursor := 0
	for i := range lineNumber {
		cursor += e.lineMap[i]
	}
	lineBytes := e.buf.Read(cursor, e.lineMap[lineNumber])
	runes := make([]rune, 0, utf8.RuneCount(lineBytes))
	for i := 0; i < len(lineBytes); {
		r, size := utf8.DecodeRune(lineBytes[i:])
		runes = append(runes, r)
		i += size
	}
	return runes
}

// TODO: improve this
func (e *Editor) getAll() []byte {
	return e.buf.ReadAll()
}

func (e *Editor) debug() {
	log.Info(fmt.Sprintf("Current line: %d, current col: %d", e.currentLine, e.currentCol))
}

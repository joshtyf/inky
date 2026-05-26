package core

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"unicode/utf8"
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
	ToggleViewMode
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

func (k KeyCode) IsEditOperation() bool {
	return k == RuneKey || k == Backspace || k == Undo
}

type Key struct {
	Code KeyCode
	Rune rune
}

type UserInterface interface {
	Start(ctx context.Context) (<-chan Key, error)
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
	LastKeyPresssed       *Key
	CursorCurrentLine     int
	CursorCurrentCol      int
	DocumentMaxLineLength int
	GetLine               func(lineNumber int) []byte
	GetAll                func() []byte
	ViewMode              ViewMode
	Version               int
}

const VERSION_MODULO = 1000000

type Editor struct {
	filePath    string
	ui          UserInterface
	buf         Buffer
	lineMap     []int
	currentLine int
	currentCol  int
	viewMode    ViewMode
	operations  []Operation
	version     int
}

func NewEditor(filePath string, ui UserInterface, buf Buffer) *Editor {
	lineMap := make([]int, 1)
	return &Editor{
		filePath:    filePath,
		ui:          ui,
		buf:         buf,
		lineMap:     lineMap,
		currentLine: 0,
		currentCol:  0,
		viewMode:    RawView,
		version:     0,
	}
}

func (e *Editor) Start(ctx context.Context) error {
	slog.Info(fmt.Sprintf("Starting Editor with file at %s", e.filePath))
	if err := e.loadFile(); err != nil {
		return err
	}
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	keyCh, err := e.ui.Start(ctx)
	if err != nil {
		return err
	}
	var lastKey *Key
	for {
		e.ui.Update(&EditorState{
			LastKeyPresssed:       lastKey,
			CursorCurrentLine:     e.currentLine,
			CursorCurrentCol:      e.currentCol,
			DocumentMaxLineLength: len(e.lineMap),
			GetLine:               e.getLine,
			GetAll:                e.getAll,
			ViewMode:              e.viewMode,
			Version:               e.version,
		})
		select {
		case <-ctx.Done():
			return nil
		case key, ok := <-keyCh:
			if !ok {
				return nil
			}
			slog.Debug("Handling key", "key", key)
			lastKey = &key
			if err := e.handleKey(key); err != nil {
				return err
			}
		}
	}
}

func (e *Editor) loadFile() error {
	if e.filePath == "" {
		panic("editor filePath cannot be empty")
	}
	f, err := os.ReadFile(e.filePath)
	if errors.Is(err, fs.ErrNotExist) {
		if dirErr := os.MkdirAll(filepath.Dir(e.filePath), 0755); dirErr != nil {
			return fmt.Errorf("could not create missing directories: %w", dirErr)
		}
		newFile, createErr := os.Create(e.filePath)
		if createErr != nil {
			return fmt.Errorf("file does not exist and could not be created: %w", createErr)
		}
		newFile.Close()
		return nil
	}
	if err != nil {
		return err
	}
	i := 0
	for i < len(f) {
		r, size := utf8.DecodeRune(f[i:])
		if r == utf8.RuneError {
			if size == 1 {
				return fmt.Errorf("error decoding rune: invalid byte sequence %v", f[i:])
			} else {
				return fmt.Errorf("error decoding rune: empty byte sequence")
			}
		}
		e.insertRune(r)
		i += size
	}
	e.currentLine = 0
	e.currentCol = 0
	return nil
}

func (e *Editor) saveFile() error {
	if e.filePath == "" {
		panic("editor filePath cannot be empty")
	}
	f, err := os.Create(e.filePath)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = e.buf.WriteTo(f)
	return err
}

func (e *Editor) handleKey(key Key) error {
	switch {
	case key.Code == CtrlD:
		return &EditorClosedError{}
	case key.Code == ToggleViewMode:
		e.toggleViewMode()
	case key.Code == Save:
		e.saveFile()
	case key.Code == ArrowUp && e.viewMode == RawView:
		e.moveCursorUp()
	case key.Code == ArrowDown && e.viewMode == RawView:
		e.moveCursorDown()
	case key.Code == ArrowLeft && e.viewMode == RawView:
		e.moveCursorLeft()
	case key.Code == ArrowRight && e.viewMode == RawView:
		e.moveCursorRight()
	case key.Code.IsEditOperation() && e.viewMode == RawView:
		op := e.getOperationFromKey(key)
		e.applyOperation(op)
		e.recordOperation(op)
	}
	return nil
}

func (e *Editor) getOperationFromKey(key Key) Operation {
	if !key.Code.IsEditOperation() {
		panic(fmt.Sprintf("key code %d is not an edit operation", key.Code))
	}
	var op Operation
	switch key.Code {
	case RuneKey:
		op = NewInsertOperation(e.getCursorPosition(), key.Rune)
	case Backspace:
		if e.getCursorPosition() == 0 {
			op = NewNoOp()
		} else {
			op = NewDeleteOperation(e.getCursorPosition()-1, e.buf.GetRune(e.getCursorPosition()-1))
		}
	case Undo:
		op = e.popLastOperation().Invert()
	}
	return op
}

func (e *Editor) applyOperation(op Operation) {
	op.Apply(e)
	e.version = (e.version + 1) % VERSION_MODULO
}

func (e *Editor) getLastOperation() Operation {
	if len(e.operations) == 0 {
		return NewNoOp()
	}
	return e.operations[len(e.operations)-1]
}

func (e *Editor) popLastOperation() Operation {
	if len(e.operations) == 0 {
		return NewNoOp()
	}
	op := e.operations[len(e.operations)-1]
	e.operations = e.operations[:len(e.operations)-1]
	return op
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
	for line < len(e.lineMap)-1 && cursor >= e.lineMap[line] {
		cursor -= e.lineMap[line]
		line++
	}
	e.currentLine = line
	e.currentCol = cursor
}

func (e *Editor) insertRune(r rune) {
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

func (e *Editor) getLine(lineNumber int) []byte {
	if lineNumber < 0 {
		panic(fmt.Sprintf("lineNumber %d is out of bounds", lineNumber))
	}
	if lineNumber >= len(e.lineMap) {
		return []byte{}
	}
	cursor := 0
	for i := range lineNumber {
		cursor += e.lineMap[i]
	}
	return e.buf.Read(cursor, e.lineMap[lineNumber])
}

func (e *Editor) getAll() []byte {
	return e.buf.ReadAll()
}

func (e *Editor) recordOperation(op Operation) {
	if op == nil {
		return
	}
	if _, ok := op.(InvertedOperation); ok {
		return
	}
	if _, ok := op.(NoOp); ok {
		return
	}
	if len(e.operations) == 0 {
		e.operations = append(e.operations, op)
	} else {
		lastOp := e.operations[len(e.operations)-1]
		if mergedOp, ok := lastOp.Merge(op); ok {
			e.operations[len(e.operations)-1] = mergedOp
		} else {
			e.operations = append(e.operations, op)
		}
	}
}

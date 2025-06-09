package core

import (
	"context"
	"log"

	editorLog "github.com/joshtyf/texteditor/log"
)

type ReadEditorLines func(start int, n int) ([]string, error)

type EditorState struct {
	KeyPressed    *Key
	CurrentLine   int
	CurrentColumn int
	ReadEditorLines
	Error error
}

type EditorIO interface {
	StartIO() (<-chan *Key, error)
	SetDisplay(es *EditorState) error
	Close() error
}

type Editor struct {
	lines         []int
	currentLine   int
	currentColumn int
	buf           Buffer
	listeners     []chan<- *EditorState
	logger        *log.Logger
	io            EditorIO
}

func NewEditor(io EditorIO, buf Buffer) *Editor {
	// TODO: proper initialization with buffer
	return &Editor{
		lines:         make([]int, 1),
		currentLine:   0,
		currentColumn: 0,
		buf:           buf,
		listeners:     make([]chan<- *EditorState, 0),
		logger:        editorLog.CreateLogger("editor"),
		io:            io,
	}
}

func (e *Editor) RegisterListener(ch chan<- *EditorState) {
	e.listeners = append(e.listeners, ch)
}

func (e *Editor) Start(ctx context.Context) error {
	e.logger.Println("starting")
	inputCh, err := e.io.StartIO()
	if err != nil {
		e.logger.Printf("error starting input: %s", err)
		return ErrEditorInternal{
			message: "failed to start input",
		}
	}
	defer func() {
		if e.io.Close() != nil {
			e.logger.Println("error closing IO")
		}
	}()

	for {
		var err error = nil
		select {
		case <-ctx.Done():
			e.logger.Println("context stopped")
			return ErrEditorQuit{}
		case k := <-inputCh:
			if k == nil {
				e.logger.Println("input channel closed unexpectedly")
				return ErrEditorInternal{
					message: "input channel closed unexpectedly",
				}
			}
			switch k.Code {
			case CtrlD:
				e.logger.Println("Ctrl+D pressed")
				return ErrEditorQuit{}
			case RuneKey:
				for _, r := range k.Runes {
					data := []byte(string(r))
					for i := range data {
						err = e.insertAtCursor(data[i])
						if err != nil {
							e.logger.Println("error encountered while inserting sequence of runes")
							break
						}
					}
				}
			case ArrowUp:
				e.moveCursorUp()
			case ArrowDown:
				e.moveCursorDown()
			case ArrowLeft:
				e.moveCursorLeft()
			case ArrowRight:
				e.moveCursorRight()
			case Newline:
				err = e.insertAtCursor('\n')
			case Backspace:
				_, err = e.backspaceAtCursor()
			case Undo:
				err = e.undo()
			}
			if coreError, ok := err.(CoreError); ok && coreError.GetSeverity() == CoreErrorSeverityFatal {
				return coreError
			}
			e.io.SetDisplay(
				&EditorState{
					KeyPressed:      k,
					CurrentLine:     e.currentLine,
					CurrentColumn:   e.currentColumn,
					ReadEditorLines: e.readLines,
					Error:           err,
				},
			)
		}

	}
}

func (e *Editor) getCursor() int {
	cursor := 0
	for i := range e.currentLine {
		cursor += e.lines[i]
	}
	return cursor + min(e.lines[e.currentLine], e.currentColumn)
}

func (e *Editor) setToCursor(cursor int) error {
	if cursor < 0 {
		return ErrEditorInternal{
			message: "received negative cursor input when setting cursor",
		}
	}
	column := cursor
	for i := range e.lines {
		if column < e.lines[i] {
			e.currentLine = i
			e.currentColumn = column
			return nil
		}
		column -= e.lines[i]
	}
	if column > 0 {
		return ErrEditorInternal{
			message: "received out of bounds cursor input when setting cursor",
		}
	}
	return nil
}
func (e *Editor) insertNewLine() error {
	e.lines = append(e.lines, 0)
	// If inserting a newline, we need to shift all the lines after the current line
	if e.currentLine+1 < len(e.lines) {
		copy(e.lines[e.currentLine+2:], e.lines[e.currentLine+1:])
	}
	// Break length of current line and add it to the next line
	e.lines[e.currentLine+1] = e.lines[e.currentLine] - e.currentColumn
	e.lines[e.currentLine] = e.currentColumn
	// Position the cursor at the start of the next line
	e.currentLine++
	e.currentColumn = 0
	return nil
}

func (e *Editor) insertAtCursor(b byte) error {
	// TODO: change to insert rune?
	cursor := e.getCursor()
	err := e.buf.InsertByte(b, cursor)
	if err != nil {
		e.logger.Printf("error inserting byte into buffer: %s", err)
		return ErrEditorInsert{
			severity: CoreErrorSeverityError,
		}
	}
	e.currentColumn += 1
	e.lines[e.currentLine] += 1
	if b == '\n' {
		err := e.insertNewLine()
		if err != nil {
			e.logger.Printf("error updating lines on newline insert: %s", err)
			return ErrEditorInsert{
				severity: CoreErrorSeverityError,
			}
		}
	}
	return nil
}

func (e *Editor) moveCursorRight() error {
	if e.currentLine == len(e.lines)-1 && e.currentColumn == e.lines[e.currentLine] {
		return nil
	}
	e.currentColumn++
	if e.currentColumn >= e.lines[e.currentLine] && e.currentLine < len(e.lines)-1 {
		e.currentColumn = 0
		e.currentLine++
	}
	return nil
}

func (e *Editor) moveCursorLeft() error {
	if e.currentLine == 0 && e.currentColumn == 0 {
		return nil
	}
	e.currentColumn--
	if e.currentColumn < 0 && e.currentLine > 0 {
		e.currentLine--
		e.currentColumn = e.lines[e.currentLine] - 1
	}
	return nil
}

func (e *Editor) moveCursorUp() error {
	e.currentLine = max(e.currentLine-1, 0)
	e.currentColumn = min(e.currentColumn, e.lines[e.currentLine]-1)
	return nil
}

func (e *Editor) moveCursorDown() error {
	e.currentLine = min(e.currentLine+1, len(e.lines)-1)
	e.currentColumn = min(e.currentColumn, e.lines[e.currentLine])
	return nil
}

func (e *Editor) backspaceAtCursor() (byte, error) {
	cursor := e.getCursor()
	if cursor == 0 {
		return 0, nil
	}
	toDelete, err := e.buf.GetByte(cursor - 1)
	if err != nil {
		e.logger.Printf("error getting byte to be deleted: %s", err)
		return 0, ErrEditorDelete{
			severity: CoreErrorSeverityError,
		}
	}
	err = e.buf.DeleteByte(cursor - 1)
	if err != nil {
		e.logger.Printf("error deleting byte from buffer: %s", err)
		return 0, ErrEditorDelete{
			severity: CoreErrorSeverityError,
		}
	}

	if e.currentColumn == 0 {
		e.lines[e.currentLine-1] += e.lines[e.currentLine]
		copy(e.lines[e.currentLine:], e.lines[e.currentLine+1:])
		e.lines = e.lines[:len(e.lines)-1]
		e.currentLine--
		e.currentColumn = e.lines[e.currentLine]
	}
	e.currentColumn--
	e.lines[e.currentLine] -= 1
	return toDelete, nil
}

func (e *Editor) undo() error {
	// TODO: Implement undo functionality
	// Need to update current cursor, current line and current column
	// Need to update lines
	lastUndo, err := e.buf.Undo()
	if err != nil {
		e.logger.Printf("error undoing last change in buffer: %s", err)
		return ErrEditorUndo{
			severity: CoreErrorSeverityError,
		}
	}
	if lastUndo == nil {
		e.logger.Println("no changes to undo")
		return nil
	}
	if len(lastUndo.Data) == 0 {
		// Reset the current line and column to the last undo cursor
		err := e.setToCursor(lastUndo.Cursor)
		if err != nil {
			e.logger.Println("error setting cursor to last insert change")
			return err
		}
		// Calculate the number of lines to shift due to the undo
		linesToShift := 0
		for l, c, delta := e.currentLine, e.currentColumn, lastUndo.Length; delta > 0; {
			// Deduct from the delta from c to the end of the line
			delta -= e.lines[l] - c
			// If delta is non-negative and there are more lines, we need to shift
			if delta >= 0 && l+1 < len(e.lines) {
				linesToShift++
				c = 0
				l++
			}
		}
		if linesToShift > 0 {
			// Recalculate the length of the current line after shifting
			for i := 1; i <= linesToShift; i++ {
				e.lines[e.currentLine] += e.lines[e.currentLine+i]
			}
			// Perform the shifting
			// Safe to index cm.currentLine+1 since a non-zero linesToShift indicates
			// that there is at least one line after the current line
			copy(e.lines[e.currentLine+1:], e.lines[e.currentLine+linesToShift+1:])
			// Remove the lines that were shifted
			e.lines = e.lines[:len(e.lines)-linesToShift]
		}
		e.lines[e.currentLine] -= lastUndo.Length
	} else {
		// Reset the current line and column to the last undo cursor
		err := e.setToCursor(lastUndo.Cursor - lastUndo.Length + 1)
		if err != nil {
			e.logger.Println("error setting cursor to last delete change")
			return err
		}
		// Reshift the lines
		// Iterate in reverse order since lastUndo.Data is reversed
		for i := len(lastUndo.Data) - 1; i >= 0; i-- {
			e.lines[e.currentLine] += 1
			e.currentColumn += 1
			if lastUndo.Data[i] == '\n' {
				err := e.insertNewLine()
				if err != nil {
					e.logger.Printf("error re-inserting newline during undo operation: %s", err)
					return err
				}
			}
		}
	}
	return nil
}

func (e *Editor) readLines(start, n int) ([]string, error) {
	cursor := 0
	for i := range start {
		cursor += e.lines[i]
	}
	content := make([]string, n)
	for i := 0; i < n && cursor < e.buf.Len(); i++ {
		nextLine, err := e.buf.SeekToChar(cursor, '\n', 1)
		if err != nil {
			e.logger.Printf("error seeking next line index: %s", err)
			return nil, ErrEditorRead{
				severity: CoreErrorSeverityError,
			}
		}
		if nextLine == -1 {
			nextLine = e.buf.Len()
		}
		data, err := e.buf.Read(cursor, nextLine-cursor)
		if err != nil {
			e.logger.Printf("error reading till next line: %s", err)
			return nil, ErrEditorRead{
				severity: CoreErrorSeverityError,
			}
		}
		content[i] = string(data)
		cursor = nextLine + 1
	}

	return content, nil
}

/*
 * Errors
 */

type ErrEditorInsert struct {
	severity CoreErrorSeverity
}

func (e ErrEditorInsert) Error() string {
	return "editor: error inserting byte"
}

func (e ErrEditorInsert) GetSeverity() CoreErrorSeverity {
	return e.severity
}

type ErrEditorDelete struct {
	severity CoreErrorSeverity
}

func (e ErrEditorDelete) Error() string {
	return "editor: error deleting byte"
}

func (e ErrEditorDelete) GetSeverity() CoreErrorSeverity {
	return e.severity
}

type ErrEditorUndo struct {
	severity CoreErrorSeverity
}

func (e ErrEditorUndo) Error() string {
	return "editor: error undoing"
}

func (e ErrEditorUndo) GetSeverity() CoreErrorSeverity {
	return e.severity
}

type ErrEditorRead struct {
	severity CoreErrorSeverity
}

func (e ErrEditorRead) Error() string {
	return "editor: error reading lines"
}

func (e ErrEditorRead) GetSeverity() CoreErrorSeverity {
	return e.severity
}

type ErrEditorInternal struct {
	message string
}

func (e ErrEditorInternal) Error() string {
	return "editor: internal error"
}

func (e ErrEditorInternal) GetSeverity() CoreErrorSeverity {
	return CoreErrorSeverityFatal
}

type ErrEditorQuit struct{}

func (e ErrEditorQuit) Error() string {
	return "editor: quit"
}

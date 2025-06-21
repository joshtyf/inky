package core

import (
	"context"
	"fmt"
	"log"

	editorLog "github.com/joshtyf/texteditor/log"
)

type ReadEditorLines func(start int, n int) ([][]byte, error)

type EditorState struct {
	KeyPressed    *Key
	CurrentLine   int
	CurrentColumn int
	ReadEditorLines
}

type EditorIO interface {
	Start() (<-chan *Key, error)
	DisplayEditor(es *EditorState) error
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
	e.logger.Println("starting editor")
	inputCh, err := e.io.Start()
	if err != nil {
		return fmt.Errorf("error starting io: %w", err)
	}
	defer func() {
		if e.io.Close() != nil {
			e.logger.Println("error closing io")
		}
	}()

	for {
		select {
		case <-ctx.Done():
			e.logger.Println("editor stopped")
			return ErrEditorQuit{}
		case k := <-inputCh:
			if k == nil {
				return fmt.Errorf("input channel closed unexpectedly")
			}
			switch k.Code {
			case CtrlD:
				e.logger.Println("Ctrl+D pressed, stopping editor")
				return ErrEditorQuit{}
			case RuneKey:
				for _, r := range k.Runes {
					data := []byte(string(r))
					for i := range data {
						e.insertAtCursor(data[i])
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
				e.insertAtCursor('\n')
			case Backspace:
				_ = e.backspaceAtCursor()
			case Undo:
				e.undo()
			}

			e.io.DisplayEditor(
				&EditorState{
					KeyPressed:      k,
					CurrentLine:     e.currentLine,
					CurrentColumn:   e.currentColumn,
					ReadEditorLines: e.readLines,
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

func (e *Editor) setToCursor(cursor int) {
	if cursor < 0 {
		panic("editor: negative cursor position received")
	}
	column := cursor
	for i := range e.lines {
		if column < e.lines[i] {
			e.currentLine = i
			e.currentColumn = column
			return
		}
		column -= e.lines[i]
	}
	if column > 0 {
		panic("editor: cursor position out of bounds")
	}
}
func (e *Editor) insertNewLine() {
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
}

func (e *Editor) insertAtCursor(b byte) {
	// TODO: change to insert rune?
	cursor := e.getCursor()
	e.buf.InsertByte(b, cursor)
	e.currentColumn += 1
	e.lines[e.currentLine] += 1
	if b == '\n' {
		e.insertNewLine()
	}
}

func (e *Editor) moveCursorRight() {
	if e.currentLine == len(e.lines)-1 && e.currentColumn == e.lines[e.currentLine] {
		return
	}
	e.currentColumn++
	if e.currentColumn >= e.lines[e.currentLine] && e.currentLine < len(e.lines)-1 {
		e.currentColumn = 0
		e.currentLine++
	}
}

func (e *Editor) moveCursorLeft() {
	if e.currentLine == 0 && e.currentColumn == 0 {
		return
	}
	e.currentColumn--
	if e.currentColumn < 0 && e.currentLine > 0 {
		e.currentLine--
		e.currentColumn = e.lines[e.currentLine] - 1
	}
}

func (e *Editor) moveCursorUp() {
	e.currentLine = max(e.currentLine-1, 0)
	e.currentColumn = min(e.currentColumn, e.lines[e.currentLine]-1)
}

func (e *Editor) moveCursorDown() {
	e.currentLine = min(e.currentLine+1, len(e.lines)-1)
	e.currentColumn = min(e.currentColumn, e.lines[e.currentLine])
}

func (e *Editor) backspaceAtCursor() byte {
	cursor := e.getCursor()
	if cursor == 0 {
		return 0
	}
	toDelete := e.buf.GetByte(cursor - 1)
	e.buf.DeleteByte(cursor - 1)

	if e.currentColumn == 0 {
		e.lines[e.currentLine-1] += e.lines[e.currentLine]
		copy(e.lines[e.currentLine:], e.lines[e.currentLine+1:])
		e.lines = e.lines[:len(e.lines)-1]
		e.currentLine--
		e.currentColumn = e.lines[e.currentLine]
	}
	e.currentColumn--
	e.lines[e.currentLine] -= 1
	return toDelete
}

func (e *Editor) undo() {
	// TODO: Implement undo functionality
	// Need to update current cursor, current line and current column
	// Need to update lines
	lastUndo := e.buf.Undo()
	if lastUndo == nil {
		e.logger.Println("no changes to undo")
		return
	}
	if len(lastUndo.Data) == 0 {
		// Reset the current line and column to the last undo cursor
		e.setToCursor(lastUndo.Cursor)
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
		e.setToCursor(lastUndo.Cursor - lastUndo.Length + 1)

		// Reshift the lines
		// Iterate in reverse order since lastUndo.Data is reversed
		for i := len(lastUndo.Data) - 1; i >= 0; i-- {
			e.lines[e.currentLine] += 1
			e.currentColumn += 1
			if lastUndo.Data[i] == '\n' {
				e.insertNewLine()
			}
		}
	}
}

func (e *Editor) readLines(start, n int) ([][]byte, error) {
	cursor := 0
	for i := range start {
		cursor += e.lines[i]
	}
	content := make([][]byte, n)
	for i := 0; i < n && cursor < e.buf.Len(); i++ {
		nextLine := e.buf.SeekToChar(cursor, '\n', 1)
		if nextLine == -1 {
			nextLine = e.buf.Len()
		}
		data := e.buf.Read(cursor, nextLine-cursor)
		content[i] = data
		cursor = nextLine + 1
	}

	return content, nil
}

type ErrEditorQuit struct{}

func (e ErrEditorQuit) Error() string {
	return "editor: quit"
}

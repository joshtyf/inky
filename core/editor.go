package core

import (
	"context"
	"fmt"
	"log"
)

type ReadEditorLines func(start int, n int) ([]string, error)

type EditorState struct {
	KeyPressed    *Key
	CurrentLine   int
	CurrentColumn int
	ReadEditorLines
}

type Editor struct {
	lines         []int
	currentLine   int
	currentColumn int
	buf           Buffer
	listeners     []chan<- *EditorState
}

func NewEditor(buf Buffer) *Editor {
	// TODO: proper initialization with buffer
	return &Editor{
		lines:         make([]int, 1),
		currentLine:   0,
		currentColumn: 0,
		buf:           buf,
		listeners:     make([]chan<- *EditorState, 0),
	}
}

func (e *Editor) RegisterListener(ch chan<- *EditorState) {
	e.listeners = append(e.listeners, ch)
}

func (e *Editor) Start(ctx context.Context, input input) error {
	log.Println("Starting text editor")
	inputCh := make(chan *Key)
	inputCtx, cancelInput := context.WithCancelCause(ctx)
	go input.start(inputCtx, cancelInput, inputCh)

	for {
		select {
		case <-ctx.Done():
			log.Println("Aborting text editor")
			cancelInput(nil)
			e.close()
			return nil
		case <-inputCtx.Done():
			log.Println("Input Cancelled: ", context.Cause(inputCtx))
			e.close()
			return nil
		case k := <-inputCh:
			switch k.Code {
			case CtrlD:
				log.Println("Ctrl+D pressed")
				cancelInput(nil)
				e.close()
				return nil
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
				e.backspaceAtCursor()
			case Undo:
				e.undo()
			}
			e.notifyListeners(
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

func (e *Editor) notifyListeners(es *EditorState) {
	for _, ch := range e.listeners {
		ch <- es
	}
}

func (e *Editor) close() {
	for _, ch := range e.listeners {
		close(ch)
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
		return fmt.Errorf("cursor cannot be negative")
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
		return fmt.Errorf("cursor out of bounds")
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

func (e *Editor) insertAtCursor(b byte) {
	cursor := e.getCursor()
	err := e.buf.InsertByte(b, cursor)
	if err != nil {
		log.Fatalf("error inserting rune: %v", err)
	}
	e.currentColumn += 1
	e.lines[e.currentLine] += 1
	if b == '\n' {
		err := e.insertNewLine()
		if err != nil {
			log.Fatalf("error inserting new line: %v", err)
		}
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
	toDelete, err := e.buf.GetByte(cursor - 1)
	if err != nil {
		log.Fatalf("error getting byte: %v", err)
	}
	err = e.buf.DeleteByte(cursor - 1)
	if err != nil {
		log.Fatalf("error deleting byte: %v", err)
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
	return toDelete
}

func (e *Editor) undo() {
	// TODO: Implement undo functionality
	// Need to update current cursor, current line and current column
	// Need to update lines
	lastUndo, err := e.buf.Undo()
	if err != nil {
		log.Fatalf("error undoing: %v", err)
	}
	if len(lastUndo.Data) == 0 {
		// Reset the current line and column to the last undo cursor
		err := e.setToCursor(lastUndo.Cursor)
		if err != nil {
			log.Fatalf("error setting cursor: %v", err)
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
			log.Fatalf("error setting cursor: %v", err)
		}
		// Reshift the lines
		// Iterate in reverse order since lastUndo.Data is reversed
		for i := len(lastUndo.Data) - 1; i >= 0; i-- {
			e.lines[e.currentLine] += 1
			e.currentColumn += 1
			if lastUndo.Data[i] == '\n' {
				err := e.insertNewLine()
				if err != nil {
					log.Fatalf("error inserting new line: %v", err)
				}
			}
		}
	}
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
			log.Fatalf("error seeking to char: %v", err)
		}
		if nextLine == -1 {
			nextLine = e.buf.Len()
		}
		data, err := e.buf.Read(cursor, nextLine-cursor)
		if err != nil {
			log.Fatalf("error reading till new line: %v", err)
		}
		content[i] = string(data)
		cursor = nextLine + 1
	}

	return content, nil
}

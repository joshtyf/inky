package core

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"unicode/utf8"

	editorIO "github.com/joshtyf/texteditor/io"
	editorLog "github.com/joshtyf/texteditor/log"
)

type EditOpType int

const (
	InsertOp EditOpType = iota
	DeleteOp
)

type EditOp struct {
	Type            EditOpType
	StartLine       int
	StartLineColumn int
	EndLine         int
	EndLineColumn   int
	Data            []rune
}

type Editor struct {
	lines         []int
	currentLine   int
	currentColumn int
	buf           Buffer
	logger        *log.Logger
	io            editorIO.EditorIO
	file          string
	editHistory   []*EditOp // TODO: Limit the size of the history
	saved         bool
	charCount     int
}

func NewEditor(io editorIO.EditorIO, buf Buffer, file string) *Editor {
	return &Editor{
		lines:         make([]int, 1),
		currentLine:   0,
		currentColumn: 0,
		buf:           buf,
		logger:        editorLog.CreateLogger("editor"),
		io:            io,
		file:          file,
		editHistory:   make([]*EditOp, 0),
		saved:         true,
		charCount:     0,
	}
}

func (e *Editor) Start(ctx context.Context) error {
	e.logger.Println("starting editor")
	err := e.init()
	if err != nil {
		return fmt.Errorf("error initialising editor: %w", err)
	}
	inputCh, err := e.io.Start()
	if err != nil {
		return fmt.Errorf("error starting io: %w", err)
	}
	defer func() {
		if e.io.Close() != nil {
			e.logger.Println("error closing io")
		}
	}()

	var keyPressed *editorIO.Key
	for {
		e.io.DisplayEditor(
			&editorIO.EditorState{
				KeyPressed:      keyPressed,
				CurrentLine:     e.currentLine,
				CurrentColumn:   e.currentColumn,
				ReadEditorLines: e.readLines,
				EditorSaved:     e.saved,
				CharCount:       e.charCount,
			},
		)
		select {
		case <-ctx.Done():
			e.logger.Println("editor stopped")
			return ErrEditorQuit{}
		case keyPressed = <-inputCh:
			if keyPressed == nil {
				return fmt.Errorf("input channel closed unexpectedly")
			}
			switch keyPressed.Code {
			case editorIO.CtrlD:
				e.logger.Println("Ctrl+D pressed, stopping editor")
				return ErrEditorQuit{}
			case editorIO.RuneKey:
				e.updateEditHistory(InsertOp, keyPressed.Rune)
				e.insertAtCursor(keyPressed.Rune)
			case editorIO.ArrowUp:
				e.moveCursorUp()
			case editorIO.ArrowDown:
				e.moveCursorDown()
			case editorIO.ArrowLeft:
				e.moveCursorLeft()
			case editorIO.ArrowRight:
				e.moveCursorRight()
			case editorIO.Newline:
				e.updateEditHistory(InsertOp, '\n')
				e.insertAtCursor('\n')
			case editorIO.Backspace:
				deletedRune := e.backspaceAtCursor()
				e.updateEditHistory(DeleteOp, deletedRune)
			case editorIO.Undo:
				e.undo()
			case editorIO.Save:
				_ = e.Save()
			}
		}
	}
}

func (e *Editor) init() error {
	f, err := os.OpenFile(e.file, os.O_CREATE|os.O_RDWR, 0666)
	if err != nil {
		return fmt.Errorf("error opening file %s: %w", e.file, err)
	}
	defer f.Close()
	content, err := io.ReadAll(f)
	if err != nil {
		return fmt.Errorf("error reading file %s: %w", e.file, err)
	}
	// Initialize buffer with content
	for i := 0; i < len(content); i++ {
		r, size := utf8.DecodeRune(content[i:])
		if r == utf8.RuneError {
			if size == 1 {
				return fmt.Errorf("error decoding rune: invalid byte sequence")
			} else {
				return fmt.Errorf("error decoding rune: empty byte sequence")
			}
		}
		e.insertAtCursor(r)
		i += size - 1
	}
	// Reset the cursor to the start
	e.currentColumn = 0
	e.currentLine = 0
	return nil
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

// This function should only be called when the edit operation inserted/deleted a newline
func (e *Editor) updateEditorLines(op EditOpType) {
	if op == InsertOp {
		e.lines = append(e.lines, 0)
		// When inserting a newline, we need to shift all the lines after the current line
		if e.currentLine+1 < len(e.lines) {
			copy(e.lines[e.currentLine+2:], e.lines[e.currentLine+1:])
		}
		// Break length of current line and add it to the next line
		e.lines[e.currentLine+1] = e.lines[e.currentLine] - e.currentColumn
		e.lines[e.currentLine] = e.currentColumn
		// Position the cursor at the start of the next line
		e.currentLine++
		e.currentColumn = 0
	} else {
		// When deleting a newline, we need to merge the current line with the previous line
		e.lines[e.currentLine-1] += e.lines[e.currentLine]
		copy(e.lines[e.currentLine:], e.lines[e.currentLine+1:])
		e.lines = e.lines[:len(e.lines)-1]
		e.currentLine--
		e.currentColumn = e.lines[e.currentLine]
	}
}

func (e *Editor) insertAtCursor(r rune) {
	cursor := e.getCursor()
	e.buf.InsertRune(r, cursor)
	e.currentColumn += 1
	e.lines[e.currentLine] += 1
	e.charCount++
	if r == '\n' {
		e.updateEditorLines(InsertOp)
	}
}

func (e *Editor) updateEditHistory(op EditOpType, r rune) {
	// Note: updateEditHistory is called at different times.
	// For insert, it is called before the actual insertion.
	// For delete, it is called after the deletion.
	// This affects the editor's current line and column.
	if e.newEditNodeRequired(op) {
		e.editHistory = append(e.editHistory, &EditOp{
			Type:            op,
			StartLine:       e.currentLine,
			StartLineColumn: e.currentColumn,
			EndLine:         e.currentLine,
			EndLineColumn:   e.currentColumn,
			Data:            []rune{},
		})
	}
	lastEdit := e.editHistory[len(e.editHistory)-1]
	lastEdit.Data = append(lastEdit.Data, r)
	if r == '\n' {
		lastEdit.EndLine++
		lastEdit.EndLineColumn = 0
	} else {
		if op == InsertOp {
			lastEdit.EndLineColumn++
		} else {
			lastEdit.StartLineColumn--
		}
	}
	e.saved = false
}

func (e *Editor) newEditNodeRequired(op EditOpType) bool {
	if len(e.editHistory) == 0 {
		return true
	}
	lastOp := e.editHistory[len(e.editHistory)-1]
	if lastOp.Type != op {
		return true
	}
	lastEditedRune := lastOp.Data[len(lastOp.Data)-1]
	if lastEditedRune == '\n' || lastEditedRune == ' ' {
		return true
	}
	if op == InsertOp && (lastOp.EndLine != e.currentLine || lastOp.EndLineColumn != e.currentColumn) {
		return true
	} else if op == DeleteOp && (lastOp.StartLine != e.currentLine || lastOp.StartLineColumn != e.currentColumn) {
		return true
	}
	return false
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
	if e.currentLine == 0 {
		return
	}
	e.currentLine = e.currentLine - 1
	e.currentColumn = min(e.currentColumn, e.lines[e.currentLine]-1)
}

func (e *Editor) moveCursorDown() {
	e.currentLine = min(e.currentLine+1, len(e.lines)-1)
	e.currentColumn = min(e.currentColumn, e.lines[e.currentLine])
}

func (e *Editor) backspaceAtCursor() rune {
	cursor := e.getCursor()
	if cursor == 0 {
		return 0
	}
	deletedRune := e.buf.DeleteRune(cursor - 1)
	if e.currentColumn == 0 {
		e.updateEditorLines(DeleteOp)
	}
	e.currentColumn--
	e.lines[e.currentLine] -= 1
	e.charCount--
	return deletedRune
}

func (e *Editor) undo() {
	if len(e.editHistory) == 0 {
		e.logger.Println("no changes to undo")
		return
	}
	lastEdit := e.editHistory[len(e.editHistory)-1]
	if lastEdit.Type == InsertOp {
		e.currentColumn = lastEdit.EndLineColumn
		e.currentLine = lastEdit.EndLine
		for i := len(lastEdit.Data); i > 0; i-- {
			e.backspaceAtCursor()
		}
	} else {
		e.currentColumn = lastEdit.StartLineColumn + 1
		e.currentLine = lastEdit.StartLine
		for i := len(lastEdit.Data) - 1; i >= 0; i-- {
			e.insertAtCursor(lastEdit.Data[i])
		}
	}
	e.editHistory = e.editHistory[:len(e.editHistory)-1]
}

// TODO: change return type to rune?
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

func (e *Editor) Save() error {
	f, err := os.OpenFile(e.file, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0666)
	if err != nil {
		panic(fmt.Sprintf("error creating file %s while saving: %v", e.file, err))
	}
	defer f.Close()
	bytesWritten, err := e.buf.WriteTo(f)
	if err != nil {
		panic(fmt.Sprintf("error writing buffer to file %s while saving: %v", e.file, err))
	}
	e.logger.Printf("saved %d bytes to %s\n", bytesWritten, e.file)
	e.saved = true
	return nil
}

type ErrEditorQuit struct{}

func (e ErrEditorQuit) Error() string {
	return "editor: quit"
}

package main

import (
	"fmt"
	"log"
)

// Still not really sure what to name this struct
type CursorMgr struct {
	lines         []int
	currentLine   int
	currentColumn int
}

type CursorMgrOption func(*CursorMgr)

func WithBuffer(buffer Buffer) CursorMgrOption {
	return func(cm *CursorMgr) {
		for cursor := range buffer.Len() {
			nextLine, err := buffer.SeekToChar(cursor, '\n', 1)
			if err != nil {
				log.Fatalf("error finding next line: %v", err)
			}
			cm.lines[len(cm.lines)-1] = nextLine - cursor
			cm.lines = append(cm.lines, 0)
			cursor = nextLine
		}
	}
}

func NewCursorMgr(options ...CursorMgrOption) *CursorMgr {
	cm := &CursorMgr{
		lines:         make([]int, 1),
		currentLine:   0,
		currentColumn: 0,
	}
	for _, o := range options {
		o(cm)
	}
	return cm
}

func (cm *CursorMgr) getCursor() int {
	cursor := 0
	for i := range cm.currentLine {
		cursor += cm.lines[i]
	}
	return cursor + min(cm.lines[cm.currentLine], cm.currentColumn)
}

func (cm *CursorMgr) MoveCursorRight() {
	if cm.currentLine == len(cm.lines)-1 && cm.currentColumn == cm.lines[cm.currentLine] {
		return
	}
	cm.currentColumn++
	if cm.currentColumn >= cm.lines[cm.currentLine] && cm.currentLine < len(cm.lines)-1 {
		cm.currentColumn = 0
		cm.currentLine++
	}
}

func (cm *CursorMgr) MoveCursorLeft() {
	if cm.currentLine == 0 && cm.currentColumn == 0 {
		return
	}
	cm.currentColumn--
	if cm.currentColumn < 0 && cm.currentLine > 0 {
		cm.currentLine--
		cm.currentColumn = cm.lines[cm.currentLine] - 1
	}
}

func (cm *CursorMgr) MoveCursorUp() {
	cm.currentLine = max(cm.currentLine-1, 0)
	cm.currentColumn = min(cm.currentColumn, cm.lines[cm.currentLine]-1)
}

func (cm *CursorMgr) MoveCursorDown() {
	cm.currentLine = min(cm.currentLine+1, len(cm.lines)-1)
	cm.currentColumn = min(cm.currentColumn, cm.lines[cm.currentLine])
}

func (cm *CursorMgr) ReturnLine(buffer Buffer) string {
	cursor := cm.getCursor()
	if cursor == buffer.Len() {
		return ""
	}
	nextLine, err := buffer.SeekToChar(cursor, '\n', 1)
	if err != nil {
		log.Fatalf("error seeking to char: %v", err)
	}
	if nextLine == -1 {
		nextLine = buffer.Len()
	}
	content, err := buffer.Read(cursor, nextLine-cursor)
	if err != nil {
		log.Fatalf("error reading till new line: %v", err)
	}
	return string(content)
}

func (cm *CursorMgr) InsertAtCursor(b byte, buffer Buffer) {
	cursor := cm.getCursor()
	err := buffer.InsertByte(b, cursor)
	if err != nil {
		log.Fatalf("error inserting rune: %v", err)
	}
	cm.currentColumn += 1
	cm.lines[cm.currentLine] += 1
	if b == '\n' {
		cm.lines = append(cm.lines, 0)
		// If inserting a newline, we need to shift all the lines after the current line
		if cm.currentLine+1 < len(cm.lines) {
			copy(cm.lines[cm.currentLine+2:], cm.lines[cm.currentLine+1:])
		}
		// Break length of current line and add it to the next line
		cm.lines[cm.currentLine+1] = cm.lines[cm.currentLine] - cm.currentColumn
		cm.lines[cm.currentLine] = cm.currentColumn
		// Position the cursor at the start of the next line
		cm.currentLine++
		cm.currentColumn = 0
	}
}

func (cm *CursorMgr) BackspaceAtCursor(buffer Buffer) byte {
	cursor := cm.getCursor()
	if cursor == 0 {
		return 0
	}
	toDelete, err := buffer.GetByte(cursor - 1)
	if err != nil {
		log.Fatalf("error getting byte: %v", err)
	}
	err = buffer.DeleteByte(cursor - 1)
	if err != nil {
		log.Fatalf("error deleting byte: %v", err)
	}

	if cm.currentColumn == 0 {
		cm.lines[cm.currentLine-1] += cm.lines[cm.currentLine]
		copy(cm.lines[cm.currentLine:], cm.lines[cm.currentLine+1:])
		cm.lines = cm.lines[:len(cm.lines)-1]
		cm.currentLine--
		cm.currentColumn = cm.lines[cm.currentLine]
	}
	cm.currentColumn--
	cm.lines[cm.currentLine] -= 1
	return toDelete
}

func (cm *CursorMgr) Undo(buffer Buffer) {
	// TODO: Implement undo functionality
	// Need to update current cursor, current line and current column
	// Need to update lines
	lastUndo, err := buffer.Undo()
	if err != nil {
		log.Fatalf("error undoing: %v", err)
	}
	if lastUndo.Type == UndoInsert {
		// Reset the current line and column to the last undo cursor
		column := lastUndo.Cursor
		for i := range cm.lines {
			if column < cm.lines[i] {
				cm.currentLine = i
				cm.currentColumn = column
				break
			}
			column -= cm.lines[i]
		}
		// Calculate the number of lines to shift due to the undo
		linesToShift := 0
		for l, c, delta := cm.currentLine, cm.currentColumn, lastUndo.Length; delta > 0; {
			// Deduct from the delta from c to the end of the line
			delta -= cm.lines[l] - c
			// If delta is non-negative and there are more lines, we need to shift
			if delta >= 0 && l+1 < len(cm.lines) {
				linesToShift++
				c = 0
				l++
			}
		}
		if linesToShift > 0 {
			// Recalculate the length of the current line after shifting
			for i := 1; i <= linesToShift; i++ {
				cm.lines[cm.currentLine] += cm.lines[cm.currentLine+i]
			}
			// Perform the shifting
			// Safe to index cm.currentLine+1 since a non-zero linesToShift indicates
			// that there is at least one line after the current line
			copy(cm.lines[cm.currentLine+1:], cm.lines[cm.currentLine+linesToShift+1:])
			// Remove the lines that were shifted
			cm.lines = cm.lines[:len(cm.lines)-linesToShift]
		}
		cm.lines[cm.currentLine] -= lastUndo.Length
	}
}

func (cm *CursorMgr) GetInfo() string {
	return fmt.Sprintf("Cursor: %d(%d, %d)\nLines: %v", cm.getCursor(), cm.currentColumn, cm.currentLine, cm.lines)
}

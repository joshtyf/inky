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

func NewCursorMgr() *CursorMgr {
	return &CursorMgr{
		lines:         make([]int, 1),
		currentLine:   0,
		currentColumn: 0,
	}
}

func (cm *CursorMgr) getCursor() int {
	cursor := 0
	for i := 0; i < cm.currentLine; i++ {
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
	content, err := buffer.ReadTillNewLine(cursor)
	if err != nil {
		log.Fatalf("error reading till new line: %v", err)
	}
	return content
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
			cm.lines[cm.currentLine+1] = 0
		}
		cm.currentLine++
		cm.currentColumn = 0
	}
}

func (cm *CursorMgr) BackspaceAtCursor(buffer Buffer) {
	cursor := cm.getCursor()
	if cursor == 0 {
		return
	}
	err := buffer.DeleteByte(cursor - 1)
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
}

func (cm *CursorMgr) GetInfo() string {
	return fmt.Sprintf("Cursor: %d(%d, %d)\nLines: %v", cm.getCursor(), cm.currentColumn, cm.currentLine, cm.lines)
}

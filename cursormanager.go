package main

import (
	"fmt"
	"log"
)

// Still not really sure what to name this struct
type CursorMgr struct {
	cursor int
	buffer Buffer
}

func NewCursorMgr(buf Buffer) *CursorMgr {
	return &CursorMgr{
		cursor: 0,
		buffer: buf,
	}
}

func (cm *CursorMgr) MoveCursorRight(offset int) {
	cm.cursor = min(cm.cursor+offset, cm.buffer.Len())
}

func (cm *CursorMgr) MoveCursorLeft(offset int) {
	cm.cursor = max(cm.cursor-offset, 0)
}

func (cm *CursorMgr) MoveCursorUp(offset int) {
	for i := 0; i < offset; i++ {
		startOfLine, err := cm.buffer.FindPrevLine(cm.cursor)
		if err != nil {
			log.Fatalf("error seeking to start of line: %v", err)
		}
		cm.cursor = startOfLine
	}
}

func (cm *CursorMgr) MoveCursorDown(offset int) {
	for i := 0; i < offset; i++ {
		endOfLine, err := cm.buffer.FindNextLine(cm.cursor)
		if err != nil {
			log.Fatalf("error seeking to end of line: %v", err)
		}
		cm.cursor = endOfLine
	}
}

func (cm *CursorMgr) ReturnLine() string {
	content, err := cm.buffer.ReadTillNewLine(cm.cursor)
	if err != nil {
		log.Fatalf("error reading till new line: %v", err)
	}
	return content
}

func (cm *CursorMgr) InsertByte(b byte) {
	err := cm.buffer.InsertByte(b, cm.cursor)
	if err != nil {
		log.Fatalf("error inserting rune: %v", err)
	}
	cm.cursor += 1
}

func (cm *CursorMgr) BackspaceAtCursor() {
	if cm.cursor == 0 {
		return
	}
	err := cm.buffer.DeleteByte(cm.cursor - 1)
	if err != nil {
		log.Fatalf("error deleting rune: %v", err)
	}
	cm.cursor -= 1
}

func (cm *CursorMgr) GetInfo() string {
	return fmt.Sprintf("Cursor: %d", cm.cursor)
}

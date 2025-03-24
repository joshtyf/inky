package main

import (
	"fmt"
	"log"
)

const (
	DEFAULT_BUFFER_SIZE = 20
)

type ErrInvalidPosition struct {
	errPos int
}

func (e ErrInvalidPosition) Error() string {
	return fmt.Sprintf("invalid position at %d", e.errPos)
}

type GapBuffer struct {
	buffer   []byte
	gapStart int
	gapEnd   int
}

func NewGapBuffer() *GapBuffer {
	return &GapBuffer{
		buffer:   make([]byte, DEFAULT_BUFFER_SIZE),
		gapStart: 0,
		gapEnd:   DEFAULT_BUFFER_SIZE,
	}
}

func (gb *GapBuffer) cursorToBufferPos(cursor int) int {
	if cursor < gb.gapStart {
		return cursor
	}
	return cursor + gb.getGapSize()
}

func (gb *GapBuffer) bufferPosToCursor(pos int) int {
	if pos < gb.gapStart {
		return pos
	}
	return pos - gb.getGapSize()
}

func (gb *GapBuffer) shiftGapEndTo(pos int) error {
	if pos < gb.gapEnd || pos > len(gb.buffer) {
		return ErrInvalidPosition{errPos: pos}
	}
	if pos == gb.gapEnd {
		return nil
	}
	delta := pos - gb.gapEnd
	gapToMove := gb.buffer[gb.gapEnd : gb.gapEnd+delta]
	gapToFill := gb.buffer[gb.gapStart : gb.gapStart+delta]
	copy(gapToFill, gapToMove)
	clear(gapToMove)
	gb.gapStart += delta
	gb.gapEnd += delta
	return nil
}

func (gb *GapBuffer) shiftGapStartTo(pos int) error {
	if pos < 0 || pos > gb.gapStart {
		return ErrInvalidPosition{errPos: pos}
	}
	if pos == gb.gapStart {
		return nil
	}
	delta := gb.gapStart - pos
	gapToMove := gb.buffer[gb.gapStart-delta : gb.gapStart]
	gapToFill := gb.buffer[gb.gapEnd-delta : gb.gapEnd]
	copy(gapToFill, gapToMove)
	clear(gapToMove)
	gb.gapStart -= delta
	gb.gapEnd -= delta
	return nil
}

func (gb *GapBuffer) getGapSize() int {
	return gb.gapEnd - gb.gapStart
}

func (gb *GapBuffer) resizeBuffer(requiredSize int) {
	if requiredSize <= gb.getGapSize() {
		return
	}
	newSize := max(len(gb.buffer)*2, len(gb.buffer)+requiredSize)
	newBuf := make([]byte, newSize)
	sizeOfRight := len(gb.buffer) - gb.gapEnd
	copy(newBuf[:gb.gapStart], gb.buffer[:gb.gapStart])
	copy(newBuf[len(newBuf)-sizeOfRight:], gb.buffer[gb.gapEnd:])
	gb.buffer = newBuf
	gb.gapEnd = len(newBuf) - sizeOfRight
}

func (gb *GapBuffer) seekToEndOfLine(pos int) (int, error) {
	if pos < 0 || pos > len(gb.buffer) {
		return -1, ErrInvalidPosition{errPos: pos}
	}
	for i := pos; i < len(gb.buffer); i++ {
		if i == gb.gapStart {
			i = gb.gapEnd
			if i == len(gb.buffer) {
				break
			}
		}
		if gb.buffer[i] == '\n' {
			return i, nil
		}

	}
	return len(gb.buffer), nil
}

func (gb *GapBuffer) seekToStartOfLine(pos int) (int, error) {
	if pos < 0 || pos > len(gb.buffer) {
		return -1, ErrInvalidPosition{errPos: pos}
	}
	for i := pos; i >= 0; i-- {
		if gb.buffer[i] == '\n' {
			return i + 1, nil
		}
		if i == gb.gapEnd {
			i = gb.gapStart
		}
	}
	return 0, nil
}

func (gb *GapBuffer) FindNextLine(cursor int) (int, error) {
	if cursor < 0 || cursor >= gb.Len() {
		return -1, ErrInvalidPosition{errPos: cursor}
	}
	pos := gb.cursorToBufferPos(cursor)
	endPos, err := gb.seekToEndOfLine(pos)
	if err != nil {
		return -1, ErrInvalidPosition{errPos: cursor}
	}
	if endPos == len(gb.buffer) {
		return gb.Len(), nil
	}
	return gb.bufferPosToCursor(endPos + 1), nil
}

// Returns -1 if the cursor points to the first line
func (gb *GapBuffer) FindPrevLine(cursor int) (int, error) {
	if cursor < 0 || cursor >= gb.Len() {
		return -1, ErrInvalidPosition{errPos: cursor}
	}
	pos := gb.cursorToBufferPos(cursor)
	if (pos == len(gb.buffer) || gb.buffer[pos] == '\n') && pos > 0 {
		pos -= 1
	}
	startPos, err := gb.seekToStartOfLine(pos)
	if err != nil {
		return -1, ErrInvalidPosition{errPos: cursor}
	}
	return gb.bufferPosToCursor(startPos - 1), nil
}

func (gb *GapBuffer) ReadTillNewLine(cursor int) (string, error) {
	if cursor < 0 || cursor >= gb.Len() {
		return "", ErrInvalidPosition{errPos: cursor}
	}
	pos := gb.cursorToBufferPos(cursor)
	endPos, err := gb.seekToEndOfLine(pos)
	if err != nil {
		return "", ErrInvalidPosition{errPos: cursor}
	}
	return string(gb.buffer[pos:endPos]), nil
}

func (gb *GapBuffer) InsertByte(b byte, cursor int) error {
	if cursor < 0 || cursor > gb.Len()+1 { // +1 because we can insert at the end of the buffer
		return ErrInvalidPosition{errPos: cursor}
	}
	if gb.getGapSize() == 0 {
		gb.resizeBuffer(len(gb.buffer) + 1)
	}
	pos := gb.cursorToBufferPos(cursor)
	if pos < gb.gapStart {
		if err := gb.shiftGapStartTo(pos); err != nil {
			log.Printf("error shifting gap start to %d: %v", pos, err)
			return ErrInvalidPosition{errPos: cursor}
		}
	} else {
		if err := gb.shiftGapEndTo(pos); err != nil {
			log.Printf("error shifting gap end to %d: %v", pos, err)
			return ErrInvalidPosition{errPos: cursor}
		}
	}
	gb.buffer[gb.gapStart] = b
	gb.gapStart += 1
	return nil
}

func (gb *GapBuffer) DeleteByte(cursor int) error {
	if cursor < 0 || cursor >= gb.Len() {
		return ErrInvalidPosition{errPos: cursor}
	}
	pos := gb.cursorToBufferPos(cursor)
	if pos < gb.gapStart {
		if err := gb.shiftGapStartTo(pos); err != nil {
			log.Printf("error shifting gap start to %d: %v", pos, err)
			return ErrInvalidPosition{errPos: cursor}
		}
	} else {
		if err := gb.shiftGapEndTo(pos); err != nil {
			log.Printf("error shifting gap end to %d: %v", pos, err)
			return ErrInvalidPosition{errPos: cursor}
		}
	}
	clear(gb.buffer[gb.gapEnd : gb.gapEnd+1])
	gb.gapEnd += 1
	return nil
}

func (gb *GapBuffer) GetByte(cursor int) (byte, error) {
	if cursor < 0 || cursor >= gb.Len() {
		return 0, ErrInvalidPosition{errPos: cursor}
	}
	pos := gb.cursorToBufferPos(cursor)
	return gb.buffer[pos], nil
}

func (gb *GapBuffer) Len() int {
	return len(gb.buffer) - gb.getGapSize()
}

func (gb *GapBuffer) GetInfo() string {
	return fmt.Sprintf("Buffer: %v\nGap Start: %d\nGap End: %d\nGap Size: %d\n", gb.buffer, gb.gapStart, gb.gapEnd, gb.getGapSize())
}

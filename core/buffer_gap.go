package core

import (
	"fmt"
)

const (
	DEFAULT_BUFFER_SIZE = 20
)

type GapBuffer struct {
	buffer       []byte
	gapStart     int
	gapEnd       int
	latestChange *ChangeNode
	undoList     []*ChangeNode
}

func NewGapBuffer() *GapBuffer {
	return &GapBuffer{
		buffer:       make([]byte, DEFAULT_BUFFER_SIZE),
		gapStart:     0,
		gapEnd:       DEFAULT_BUFFER_SIZE,
		latestChange: nil,
		undoList:     make([]*ChangeNode, 0),
	}
}

func NewGapBufferWithContent(content []byte) *GapBuffer {
	buf := make([]byte, len(content)+DEFAULT_BUFFER_SIZE)
	copy(buf, content)
	return &GapBuffer{
		buffer:   buf,
		gapStart: len(content),
		gapEnd:   len(buf),
	}
}

func (gb *GapBuffer) cursorToBufferPos(cursor int) int {
	if cursor < 0 || cursor > gb.Len() {
		panic(fmt.Sprintf("buffer: cursor out of range: %d, buffer length: %d", cursor, gb.Len()))
	}
	if cursor <= gb.gapStart {
		return cursor
	}
	return cursor + gb.getGapSize()
}

func (gb *GapBuffer) bufferPosToCursor(pos int) int {
	if pos <= gb.gapStart {
		return pos
	}
	return pos - gb.getGapSize()
}

func (gb *GapBuffer) shiftGapEndTo(pos int) {
	if pos < gb.gapEnd || pos > len(gb.buffer) {
		panic(fmt.Sprintf("buffer: position out of range: %d, gap end: %d, buffer length: %d", pos, gb.gapEnd, len(gb.buffer)))
	}
	if pos != gb.gapEnd {
		delta := pos - gb.gapEnd
		toMove := gb.buffer[gb.gapEnd : gb.gapEnd+delta]
		toFill := gb.buffer[gb.gapStart : gb.gapStart+delta]
		copy(toFill, toMove)
		clear(gb.buffer[pos-gb.getGapSize() : pos])
		gb.gapStart += delta
		gb.gapEnd += delta
	}
}

func (gb *GapBuffer) shiftGapStartTo(pos int) {
	if pos < 0 || pos > gb.gapStart {
		panic(fmt.Sprintf("buffer: position out of range: %d, gap start: %d, buffer length: %d", pos, gb.gapStart, len(gb.buffer)))
	}
	if pos != gb.gapStart {
		delta := gb.gapStart - pos
		toMove := gb.buffer[gb.gapStart-delta : gb.gapStart]
		toFill := gb.buffer[gb.gapEnd-delta : gb.gapEnd]
		copy(toFill, toMove)
		clear(gb.buffer[pos : pos+gb.getGapSize()])
		gb.gapStart -= delta
		gb.gapEnd -= delta
	}
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

func (gb *GapBuffer) save() {
	if gb.latestChange != nil {
		if len(gb.latestChange.Data) > 0 && len(gb.latestChange.Data) != gb.latestChange.Length {
			panic(fmt.Sprintf("buffer: save data length mismatch: expected %d, got %d instead", len(gb.latestChange.Data), gb.latestChange.Length))
		}
		gb.undoList = append(gb.undoList, gb.latestChange)
		gb.latestChange = nil
	}
}

func (gb *GapBuffer) SeekToChar(cursor int, char byte, count int) int {
	// TODO: add a test case to account for when i jumps over the gap
	if count <= 0 {
		panic(fmt.Sprintf("buffer: expect positive count, got %d instead", count))
	}
	pos := gb.cursorToBufferPos(cursor)
	for i, dist := pos, 0; i < len(gb.buffer); i, dist = i+1, dist+1 {
		if i == gb.gapStart {
			i = gb.gapEnd
			if i == len(gb.buffer) {
				break
			}
		}
		if gb.buffer[i] == char {
			count--
			if count == 0 {
				return cursor + dist
			}
		}
	}
	return -1
}

func (gb *GapBuffer) ReverseSeekToChar(cursor int, char byte, count int) int {
	if count <= 0 {
		panic(fmt.Sprintf("buffer: expect positive count, got %d instead", count))
	}
	pos := gb.cursorToBufferPos(cursor)
	for i, dist := pos, 0; i >= 0; i, dist = i-1, dist+1 {
		if i == gb.gapEnd {
			i = gb.gapStart
			if i < 0 {
				break
			}
		}
		if gb.buffer[i] == char {
			count--
			if count == 0 {
				return cursor - dist
			}
		}
	}
	return -1
}

func (gb *GapBuffer) Read(cursor int, length int) []byte {
	if length < 0 {
		panic(fmt.Sprintf("buffer: negative length %d received", length))
	}
	pos := gb.cursorToBufferPos(cursor)
	contents := make([]byte, 0)
	for i := pos; i < len(gb.buffer) && len(contents) < length; i++ {
		if i == gb.gapStart {
			i = gb.gapEnd
			if i >= len(gb.buffer) {
				break
			}
		}
		contents = append(contents, gb.buffer[i])
	}
	return contents
}

func (gb *GapBuffer) InsertByte(b byte, cursor int) {
	if gb.getGapSize() == 0 {
		gb.resizeBuffer(len(gb.buffer) + 1)
	}
	if gb.latestChange == nil || cursor != gb.latestChange.Cursor+gb.latestChange.Length || len(gb.latestChange.Data) > 0 {
		gb.save()
		gb.latestChange = &ChangeNode{
			Cursor: cursor,
			Length: 0,
			Data:   make([]byte, 0),
		}
	}
	pos := gb.cursorToBufferPos(cursor)
	if pos <= gb.gapStart {
		gb.shiftGapStartTo(pos)
	} else {
		gb.shiftGapEndTo(pos)
	}
	gb.buffer[gb.gapStart] = b
	gb.gapStart += 1
	gb.latestChange.Length += 1
	// If inserted byte is a whitespace or newline, we need to save the buffer
	if b == ' ' || b == '\n' {
		gb.save()
		gb.latestChange = &ChangeNode{
			Cursor: cursor + 1, // Next change will start after the inserted byte
			Length: 0,
			Data:   make([]byte, 0),
		}
	}
}

func (gb *GapBuffer) DeleteByte(cursor int) {
	if gb.Len() == 0 {
		panic("buffer: cannot delete byte from an empty buffer")
	}
	if cursor >= gb.Len() {
		panic(fmt.Sprintf("buffer: cursor out of range: %d, buffer length: %d", cursor, gb.Len()))
	}
	if gb.latestChange == nil || cursor != gb.latestChange.Cursor-gb.latestChange.Length || len(gb.latestChange.Data) == 0 {
		gb.save()
		gb.latestChange = &ChangeNode{
			Cursor: cursor,
			Length: 0,
			Data:   make([]byte, 0),
		}
	}
	pos := gb.cursorToBufferPos(cursor)
	if pos <= gb.gapStart {
		gb.shiftGapStartTo(pos)
	} else {
		gb.shiftGapEndTo(pos)
	}
	byteToDelete := gb.buffer[gb.gapEnd]
	clear(gb.buffer[gb.gapEnd : gb.gapEnd+1])
	gb.gapEnd += 1
	gb.latestChange.Length += 1
	gb.latestChange.Data = append(gb.latestChange.Data, byteToDelete)
}

func (gb *GapBuffer) GetByte(cursor int) byte {
	if cursor < 0 || cursor >= gb.Len() {
		panic(fmt.Sprintf("buffer: cursor out of range: %d, buffer length: %d", cursor, gb.Len()))
	}
	pos := gb.cursorToBufferPos(cursor)
	return gb.buffer[pos]
}

func (gb *GapBuffer) Undo() *ChangeNode {
	// TODO: check if undo will be affected by buffer resize (I don't think so)
	// Save the current state before undoing
	gb.save()
	if len(gb.undoList) == 0 {
		return nil
	}
	lastUndo := gb.undoList[len(gb.undoList)-1]
	gb.undoList = gb.undoList[:len(gb.undoList)-1]
	var undoPos int
	if len(lastUndo.Data) > 0 {
		// Seek to the position of the last deleted byte
		undoPos = gb.cursorToBufferPos(lastUndo.Cursor - lastUndo.Length + 1)
	} else {
		// Seek to the position of the first inserted byte
		undoPos = gb.cursorToBufferPos(lastUndo.Cursor)
	}
	if undoPos <= gb.gapStart {
		gb.shiftGapStartTo(undoPos)
	} else {
		gb.shiftGapEndTo(undoPos)
	}
	if len(lastUndo.Data) > 0 {
		// lastUndo.Data needs to inserted in reverse order
		for i := len(lastUndo.Data) - 1; i >= 0; i-- {
			gb.buffer[gb.gapStart] = lastUndo.Data[i]
			gb.gapStart += 1
		}
	} else {
		// Delete the bytes of the last undo
		// First shift the gap so that gap end is at the start of the last undo
		clear(gb.buffer[gb.gapEnd : gb.gapEnd+lastUndo.Length])
		gb.gapEnd += lastUndo.Length
	}
	gb.latestChange = nil
	return lastUndo
}

func (gb *GapBuffer) Len() int {
	return len(gb.buffer) - gb.getGapSize()
}

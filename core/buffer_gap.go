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

func (gb *GapBuffer) cursorToBufferPos(cursor int) (int, error) {
	if cursor < 0 || cursor > gb.Len() {
		return -1, fmt.Errorf("cursor out of range: %d", cursor)
	}
	if cursor <= gb.gapStart {
		return cursor, nil
	}
	return cursor + gb.getGapSize(), nil
}

func (gb *GapBuffer) bufferPosToCursor(pos int) int {
	if pos <= gb.gapStart {
		return pos
	}
	return pos - gb.getGapSize()
}

func (gb *GapBuffer) shiftGapEndTo(pos int) error {
	if pos < gb.gapEnd || pos > len(gb.buffer) {
		return fmt.Errorf("position out of range: %d", pos)
	}
	if pos == gb.gapEnd {
		return nil
	}
	delta := pos - gb.gapEnd
	toMove := gb.buffer[gb.gapEnd : gb.gapEnd+delta]
	toFill := gb.buffer[gb.gapStart : gb.gapStart+delta]
	copy(toFill, toMove)
	clear(gb.buffer[pos-gb.getGapSize() : pos])
	gb.gapStart += delta
	gb.gapEnd += delta
	return nil
}

func (gb *GapBuffer) shiftGapStartTo(pos int) error {
	if pos < 0 || pos > gb.gapStart {
		return fmt.Errorf("position out of range: %d", pos)
	}
	if pos == gb.gapStart {
		return nil
	}
	delta := gb.gapStart - pos
	toMove := gb.buffer[gb.gapStart-delta : gb.gapStart]
	toFill := gb.buffer[gb.gapEnd-delta : gb.gapEnd]
	copy(toFill, toMove)
	clear(gb.buffer[pos : pos+gb.getGapSize()])
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

func (gb *GapBuffer) save() error {
	if gb.latestChange != nil {
		if len(gb.latestChange.Data) > 0 && len(gb.latestChange.Data) != gb.latestChange.Length {
			return fmt.Errorf("save data length mismatch: expected %d, got %d instead", len(gb.latestChange.Data), gb.latestChange.Length)
		}
		gb.undoList = append(gb.undoList, gb.latestChange)
		gb.latestChange = nil
	}
	return nil
}

func (gb *GapBuffer) SeekToChar(cursor int, char byte, count int) (int, error) {
	// TODO: add a test case to account for when i jumps over the gap
	if count <= 0 {
		return -1, fmt.Errorf("buffer: expect positive count, got %d instead", count)
	}
	pos, err := gb.cursorToBufferPos(cursor)
	if err != nil {
		return -1, fmt.Errorf("buffer: error converting cursor to buffer position when seeking to character: %s", err)
	}

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
				return cursor + dist, nil
			}
		}
	}
	return -1, nil
}

func (gb *GapBuffer) ReverseSeekToChar(cursor int, char byte, count int) (int, error) {
	if count <= 0 {
		return -1, fmt.Errorf("buffer: expect positive count, got %d instead", count)
	}
	pos, err := gb.cursorToBufferPos(cursor)
	if err != nil {
		return -1, fmt.Errorf("buffer: error converting cursor to buffer position when reverse seeking to character: %s", err)
	}
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
				return cursor - dist, nil
			}
		}
	}
	return -1, nil
}

func (gb *GapBuffer) Read(cursor int, length int) ([]byte, error) {
	if length < 0 {
		return nil, fmt.Errorf("buffer: negative length %d received", length)
	}
	pos, err := gb.cursorToBufferPos(cursor)
	if err != nil {
		return nil, fmt.Errorf("buffer: error converting cursor to buffer position when reading: %s", err)
	}
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
	return contents, nil
}

func (gb *GapBuffer) InsertByte(b byte, cursor int) error {
	if gb.getGapSize() == 0 {
		gb.resizeBuffer(len(gb.buffer) + 1)
	}
	if gb.latestChange == nil || cursor != gb.latestChange.Cursor+gb.latestChange.Length || len(gb.latestChange.Data) > 0 {
		err := gb.save()
		if err != nil {
			return fmt.Errorf("buffer: error saving before inserting byte: %s", err)
		}
		gb.latestChange = &ChangeNode{
			Cursor: cursor,
			Length: 0,
			Data:   make([]byte, 0),
		}
	}
	pos, err := gb.cursorToBufferPos(cursor)
	if err != nil {
		return fmt.Errorf("buffer: error converting cursor to buffer position when getting byte: %s", err)
	}
	if pos <= gb.gapStart {
		if err := gb.shiftGapStartTo(pos); err != nil {
			return fmt.Errorf("buffer: error shifting gap start to %d when inserting byte, %s", pos, err)
		}
	} else {
		if err := gb.shiftGapEndTo(pos); err != nil {
			return fmt.Errorf("buffer: error shifting gap end to %d when inserting byte, %s", pos, err)
		}
	}
	gb.buffer[gb.gapStart] = b
	gb.gapStart += 1
	gb.latestChange.Length += 1
	// If inserted byte is a whitespace or newline, we need to save the buffer
	if b == ' ' || b == '\n' {
		err := gb.save()
		if err != nil {
			return fmt.Errorf("buffer: error saving after whitespace or newline inserted: %s", err)
		}
		gb.latestChange = &ChangeNode{
			Cursor: cursor + 1, // Next change will start after the inserted byte
			Length: 0,
			Data:   make([]byte, 0),
		}
	}
	return nil
}

func (gb *GapBuffer) DeleteByte(cursor int) error {
	if gb.Len() == 0 {
		return fmt.Errorf("buffer: cannot delete byte from an empty buffer")
	}
	if cursor >= gb.Len() {
		return fmt.Errorf("buffer: cursor must be on a non-empty buffer position, got %d instead", cursor)
	}
	if gb.latestChange == nil || cursor != gb.latestChange.Cursor-gb.latestChange.Length || len(gb.latestChange.Data) == 0 {
		err := gb.save()
		if err != nil {
			return fmt.Errorf("buffer: error saving before deleting byte: %s", err)
		}
		gb.latestChange = &ChangeNode{
			Cursor: cursor,
			Length: 0,
			Data:   make([]byte, 0),
		}
	}
	pos, err := gb.cursorToBufferPos(cursor)
	if err != nil {
		return fmt.Errorf("buffer: error converting cursor to buffer position when getting byte: %s", err)
	}
	if pos <= gb.gapStart {
		if err := gb.shiftGapStartTo(pos); err != nil {
			return fmt.Errorf("buffer: error shifting gap start to %d when deleting byte: %s", pos, err)
		}
	} else {
		if err := gb.shiftGapEndTo(pos); err != nil {
			return fmt.Errorf("buffer: error shifting gap end to %d when deleting byte: %s", pos, err)
		}
	}
	byteToDelete := gb.buffer[gb.gapEnd]
	clear(gb.buffer[gb.gapEnd : gb.gapEnd+1])
	gb.gapEnd += 1
	gb.latestChange.Length += 1
	gb.latestChange.Data = append(gb.latestChange.Data, byteToDelete)
	return nil
}

func (gb *GapBuffer) GetByte(cursor int) (byte, error) {
	if cursor < 0 || cursor >= gb.Len() {
		return 0, fmt.Errorf("buffer: cursor out of range: %d", cursor)
	}
	pos, err := gb.cursorToBufferPos(cursor)
	if err != nil {
		return 0, fmt.Errorf("buffer: error converting cursor to buffer position when getting byte: %s", err)
	}
	return gb.buffer[pos], nil
}

func (gb *GapBuffer) Undo() (*ChangeNode, error) {
	// TODO: check if undo will be affected by buffer resize (I don't think so)
	// Save the current state before undoing
	err := gb.save()
	if err != nil {
		return nil, fmt.Errorf("buffer: error saving before undoing: %s", err)
	}
	if len(gb.undoList) == 0 {
		return nil, nil
	}
	lastUndo := gb.undoList[len(gb.undoList)-1]
	gb.undoList = gb.undoList[:len(gb.undoList)-1]
	var undoPos int
	if len(lastUndo.Data) > 0 {
		// Seek to the position of the last deleted byte
		undoPos, err = gb.cursorToBufferPos(lastUndo.Cursor - lastUndo.Length + 1)
		if err != nil {
			return nil, fmt.Errorf("buffer: error converting cursor to buffer position when undoing a delete: %s", err)
		}
	} else {
		// Seek to the position of the first inserted byte
		undoPos, err = gb.cursorToBufferPos(lastUndo.Cursor)
		if err != nil {
			return nil, fmt.Errorf("buffer: error converting cursor to buffer position when undoing an insert: %s", err)
		}
	}
	if undoPos <= gb.gapStart {
		if err := gb.shiftGapStartTo(undoPos); err != nil {
			return nil, fmt.Errorf("buffer: error shifting gap start to %d when undoing: %s", undoPos, err)
		}
	} else {
		if err := gb.shiftGapEndTo(undoPos); err != nil {
			return nil, fmt.Errorf("buffer: error shifting gap end to %d when undoing: %s", undoPos, err)
		}
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
	return lastUndo, nil
}

func (gb *GapBuffer) Len() int {
	return len(gb.buffer) - gb.getGapSize()
}

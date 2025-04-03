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
	buffer      []byte
	gapStart    int
	gapEnd      int
	bufSaved    bool
	changeStart int
	changeLen   int
	undoList    []*UndoNode
}

func NewGapBuffer() *GapBuffer {
	return &GapBuffer{
		buffer:      make([]byte, DEFAULT_BUFFER_SIZE),
		gapStart:    0,
		gapEnd:      DEFAULT_BUFFER_SIZE,
		bufSaved:    true,
		changeStart: 0,
		changeLen:   0,
		undoList:    make([]*UndoNode, 0),
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

func (gb *GapBuffer) shiftGapEndTo(pos int) error {
	if pos < gb.gapEnd || pos > len(gb.buffer) {
		return ErrInvalidPosition{errPos: pos}
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
		return ErrInvalidPosition{errPos: pos}
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
	if !gb.bufSaved {
		gb.bufSaved = true
		gb.undoList = append(gb.undoList, &UndoNode{
			Type:   UndoInsert, // TODO: save for delete as well
			Cursor: gb.changeStart,
			Length: gb.changeLen,
			Data:   []byte{},
		})
	}
	return nil
}

func (gb *GapBuffer) SeekToChar(cursor int, char byte, count int) (int, error) {
	if cursor < 0 || cursor > gb.Len() {
		return -1, ErrInvalidPosition{errPos: cursor}
	}
	if count <= 0 {
		return -1, fmt.Errorf("count must be greater than 0")
	}
	pos := gb.cursorToBufferPos(cursor)
	for i := pos; i < len(gb.buffer); i++ {
		if i == gb.gapStart {
			i = gb.gapEnd
			if i == len(gb.buffer) {
				break
			}
		}
		if gb.buffer[i] == char {
			count--
			if count == 0 {
				return i, nil
			}
		}
	}
	return -1, nil
}

func (gb *GapBuffer) ReverseSeekToChar(cursor int, char byte, count int) (int, error) {
	if cursor < 0 || cursor > gb.Len() {
		return -1, ErrInvalidPosition{errPos: cursor}
	}
	if count <= 0 {
		return -1, fmt.Errorf("count must be greater than 0")
	}
	pos := gb.cursorToBufferPos(cursor)
	for i := pos; i >= 0; i-- {
		if i == gb.gapEnd {
			i = gb.gapStart
			if i < 0 {
				break
			}
		}
		if gb.buffer[i] == char {
			count--
			if count == 0 {
				return i, nil
			}
		}
	}
	return -1, nil
}

func (gb *GapBuffer) Read(cursor int, length int) ([]byte, error) {
	if cursor < 0 || cursor > gb.Len() {
		return nil, ErrInvalidPosition{errPos: cursor}
	}
	if length < 0 {
		return nil, fmt.Errorf("length must be greater than 0")
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
	return contents, nil
}

func (gb *GapBuffer) InsertByte(b byte, cursor int) error {
	if cursor < 0 || cursor > gb.Len() {
		return ErrInvalidPosition{errPos: cursor}
	}
	if gb.getGapSize() == 0 {
		gb.resizeBuffer(len(gb.buffer) + 1)
	}
	if cursor != gb.changeStart+gb.changeLen {
		gb.save()
		gb.changeStart = cursor
		gb.changeLen = 0
	}
	pos := gb.cursorToBufferPos(cursor)
	if pos <= gb.gapStart {
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
	gb.changeLen += 1
	gb.bufSaved = false
	// If inserted byte is a whitespace or newline, we need to save the buffer
	if b == ' ' || b == '\n' {
		gb.save()
		gb.changeStart = cursor + 1 // Next change will start after the inserted byte
		gb.changeLen = 0
	}
	return nil
}

func (gb *GapBuffer) DeleteByte(cursor int) error {
	if cursor < 0 || cursor >= gb.Len() { // TODO: Is this correct?
		return ErrInvalidPosition{errPos: cursor}
	}
	pos := gb.cursorToBufferPos(cursor)
	if pos <= gb.gapStart {
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
	if cursor < 0 || cursor >= gb.Len() { // TODO: Is this correct?
		return 0, ErrInvalidPosition{errPos: cursor}
	}
	pos := gb.cursorToBufferPos(cursor)
	return gb.buffer[pos], nil
}

func (gb *GapBuffer) Undo() (*UndoNode, error) {
	// TODO: check if undo will be affected by buffer resize (I don't think so)
	// Save the current state before undoing
	gb.save()
	if len(gb.undoList) == 0 {
		// TODO: if there are no undoes, should we throw an error?
		return nil, fmt.Errorf("nothing to undo")
	}
	lastUndo := gb.undoList[len(gb.undoList)-1]
	gb.undoList = gb.undoList[:len(gb.undoList)-1]
	// Delete the bytes of the last undo
	// First shift the gap so that gap end is at the start of the last undo
	undoPos := gb.cursorToBufferPos(lastUndo.Cursor) // TODO: check if the UndoNode should contain Cursor or bufferPos
	if undoPos <= gb.gapStart {
		if err := gb.shiftGapStartTo(undoPos); err != nil {
			log.Printf("error shifting gap start to %d: %v", undoPos, err)
			return nil, ErrInvalidPosition{errPos: undoPos}
		}
	} else {
		if err := gb.shiftGapEndTo(undoPos); err != nil {
			log.Printf("error shifting gap end to %d: %v", undoPos, err)
			return nil, ErrInvalidPosition{errPos: undoPos}
		}
	}
	clear(gb.buffer[gb.gapEnd : gb.gapEnd+lastUndo.Length])
	// TODO: check again for the correctness of the below. What happens if its a undo delete?
	gb.gapEnd += lastUndo.Length
	gb.changeStart = lastUndo.Cursor
	gb.changeLen = 0
	return lastUndo, nil
}

func (gb *GapBuffer) Len() int {
	return len(gb.buffer) - gb.getGapSize()
}

func (gb *GapBuffer) GetInfo() string {
	return fmt.Sprintf("Buffer: %v\nUndo List: %v\nChange Start: %d, Change Len: %d, Buffer saved: %t", gb.buffer, gb.undoList, gb.changeStart, gb.changeLen, gb.bufSaved)
}

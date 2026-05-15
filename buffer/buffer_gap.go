package buffer

import (
	"fmt"
	"io"
	"unicode/utf8"
)

const (
	DEFAULT_BUFFER_SIZE = 20
)

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
	if cursor < 0 || cursor > gb.Len() {
		panic(fmt.Sprintf("buffer: cursor out of range: %d, buffer length: %d", cursor, gb.Len()))
	}
	if cursor < gb.gapStart {
		return cursor
	}
	return cursor + gb.getGapSize()
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

func (gb *GapBuffer) SeekToChar(cursor int, char byte, count int) int {
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
			// jump to just before the gap; counteract the post-decrement so dist
			// stays accurate (the gap bytes are not content)
			i = gb.gapStart
			dist--
			continue
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
	contents := make([]byte, 0, length)
	if pos < gb.gapStart {
		beforeGapEnd := min(gb.gapStart, pos+length)
		contents = append(contents, gb.buffer[pos:beforeGapEnd]...)
		remaining := length - (beforeGapEnd - pos)
		if remaining > 0 {
			afterGapEnd := min(gb.gapEnd+remaining, len(gb.buffer))
			contents = append(contents, gb.buffer[gb.gapEnd:afterGapEnd]...)
		}
	} else {
		end := min(pos+length, len(gb.buffer))
		contents = append(contents, gb.buffer[pos:end]...)
	}
	return contents
}

func (gb *GapBuffer) ReadAll() []byte {
	contents := make([]byte, gb.Len())
	copy(contents[:gb.gapStart], gb.buffer[:gb.gapStart])
	copy(contents[gb.gapStart:], gb.buffer[gb.gapEnd:])
	return contents
}

func (gb *GapBuffer) InsertRune(r rune, cursor int) {
	var rawBytes [utf8.UTFMax]byte
	size := utf8.EncodeRune(rawBytes[:], r)
	if gb.getGapSize() < size {
		gb.resizeBuffer(size)
	}
	pos := gb.cursorToBufferPos(cursor)
	if pos <= gb.gapStart {
		gb.shiftGapStartTo(pos)
	} else {
		gb.shiftGapEndTo(pos)
	}
	copy(gb.buffer[gb.gapStart:gb.gapStart+size], rawBytes[:size])
	gb.gapStart += size
}

func (gb *GapBuffer) DeleteRune(cursor int) rune {
	if gb.Len() == 0 {
		panic("buffer: cannot delete rune from an empty buffer")
	}
	if cursor >= gb.Len() {
		panic(fmt.Sprintf("buffer: cursor out of range: %d, buffer length: %d", cursor, gb.Len()))
	}
	pos := gb.cursorToBufferPos(cursor)
	if pos <= gb.gapStart {
		gb.shiftGapStartTo(pos)
	} else {
		gb.shiftGapEndTo(pos)
	}
	r, size := utf8.DecodeRune(gb.buffer[gb.gapEnd:])
	if r == utf8.RuneError {
		if size == 1 {
			panic(fmt.Sprintf("buffer: error decoding rune: invalid byte sequence %v", gb.buffer[gb.gapEnd:]))
		} else {
			panic("buffer: error decoding rune: empty byte sequence")
		}
	}
	clear(gb.buffer[gb.gapEnd : gb.gapEnd+size])
	gb.gapEnd += size
	return r
}

func (gb *GapBuffer) GetRune(cursor int) rune {
	if cursor < 0 || cursor >= gb.Len() {
		panic(fmt.Sprintf("buffer: cursor out of range: %d, buffer length: %d", cursor, gb.Len()))
	}
	pos := gb.cursorToBufferPos(cursor)
	var byteSequence []byte
	if pos < gb.gapStart {
		byteSequence = gb.buffer[pos:gb.gapStart]
	} else {
		byteSequence = gb.buffer[pos:]
	}
	r, size := utf8.DecodeRune(byteSequence)
	if r == utf8.RuneError {
		if size == 1 {
			panic(fmt.Sprintf("buffer: error decoding rune: invalid byte sequence %v", byteSequence))
		} else {
			panic("buffer: error decoding rune: empty byte sequence")
		}
	}
	return r
}

func (gb *GapBuffer) Len() int {
	return len(gb.buffer) - gb.getGapSize()
}

func (gb *GapBuffer) WriteTo(w io.Writer) (int64, error) {
	totalWritten := int64(0)
	n, err := w.Write(gb.buffer[:gb.gapStart])
	if err != nil {
		return totalWritten, fmt.Errorf("buffer: error writing to writer: %w", err)
	}
	totalWritten += int64(n)
	n, err = w.Write(gb.buffer[gb.gapEnd:])
	if err != nil {
		return totalWritten, fmt.Errorf("buffer: error writing to writer: %w", err)
	}
	totalWritten += int64(n)
	return totalWritten, nil
}

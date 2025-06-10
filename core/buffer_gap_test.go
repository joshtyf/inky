package core

import (
	"errors"
	"fmt"
	"testing"
)

func TestShiftGapStart_ExpectErrors(t *testing.T) {
	tests := []struct {
		name          string
		inputPosition int
		expectedErr   error
	}{
		{"position less than 0", -1, errors.New("position out of range: -1")},
		{"position greater than gap start", 1, errors.New("position out of range: 1")},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			gb := NewGapBuffer()
			err := gb.shiftGapEndTo(test.inputPosition)
			if err == nil || err.Error() != test.expectedErr.Error() {
				t.Errorf("expected error %v, got %v", test.expectedErr, err)
			}
		})
	}
}

func TestShiftGapEnd_ExpectErrors(t *testing.T) {
	tests := []struct {
		name          string
		inputPosition int
		expectedErr   error
	}{
		{"position less than gap end", DEFAULT_BUFFER_SIZE - 1, fmt.Errorf("position out of range: %d", DEFAULT_BUFFER_SIZE-1)},
		{"position greater than buffer size", DEFAULT_BUFFER_SIZE + 1, fmt.Errorf("position out of range: %d", DEFAULT_BUFFER_SIZE+1)},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			gb := NewGapBuffer()
			err := gb.shiftGapEndTo(test.inputPosition)
			if err == nil || err.Error() != test.expectedErr.Error() {
				t.Errorf("expected error %v, got %v", test.expectedErr, err)
			}
		})
	}
}

func TestSeekToChar_NoErrors(t *testing.T) {
	tests := []struct {
		name     string
		input    []byte
		cursor   int
		char     byte
		count    int
		expected int
	}{
		{"find first occurrence with cursor before first", []byte("hello world"), 0, 'o', 1, 4},
		{"find second occurrence with cursor before first", []byte("hello world"), 0, 'o', 2, 7},
		{"find third occurrence with cursor before first", []byte("hello world"), 0, 'o', 3, -1},
		{"find first occurrence with cursor at first", []byte("hello world"), 4, 'o', 1, 4},
		{"find second occurrence with cursor at first", []byte("hello world"), 4, 'o', 2, 7},
		{"find second occurence with cursor after first", []byte("hello world"), 5, 'o', 1, 7},
		{"find third occurence with cursor after first", []byte("hello world"), 5, 'o', 2, -1},
		{"find unknown character", []byte("hello world"), 0, 'p', 1, -1},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			gb := NewGapBufferWithContent(test.input)
			result, err := gb.SeekToChar(test.cursor, test.char, test.count)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if result != test.expected {
				t.Errorf("expected %d, got %d", test.expected, result)
			}
		})
	}
}

func TestSeekToChar_WithErrors(t *testing.T) {
	gb := NewGapBufferWithContent([]byte("hello world"))
	tests := []struct {
		name     string
		cursor   int
		char     byte
		count    int
		expected error
	}{
		{"negative cursor", -1, 'o', 1, errors.New("buffer: error converting cursor to buffer position when seeking to character: cursor out of range: -1")},
		{"cursor greater than buffer content length", gb.Len() + 1, 'o', 1, errors.New("buffer: error converting cursor to buffer position when seeking to character: cursor out of range: 12")},
		{"cursor greater than buffer length", len(gb.buffer) + 1, 'o', 1, errors.New("buffer: error converting cursor to buffer position when seeking to character: cursor out of range: 32")},
		{"zero count", 0, 'o', 0, errors.New("buffer: expect positive count, got 0 instead")},
		{"negative count", 0, 'o', -1, errors.New("buffer: expect positive count, got -1 instead")},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result, err := gb.SeekToChar(test.cursor, test.char, test.count)
			if err == nil || err.Error() != test.expected.Error() {
				t.Errorf("expected error %v, got %v", test.expected, err)
			}
			if result != -1 {
				t.Errorf("expected -1 result, got %d", result)
			}
		})
	}
}

func TestReverseSeekToChar_NoErrors(t *testing.T) {
	tests := []struct {
		name     string
		input    []byte
		cursor   int
		char     byte
		count    int
		expected int
	}{
		{"find first occurrence with cursor before first", []byte("hello world"), 11, 'o', 1, 7},
		{"find second occurrence with cursor before first", []byte("hello world"), 11, 'o', 2, 4},
		{"find third occurrence with cursor before first", []byte("hello world"), 11, 'o', 3, -1},
		{"find first occurrence with cursor at first", []byte("hello world"), 7, 'o', 1, 7},
		{"find second occurence with cursor at first", []byte("hello world"), 7, 'o', 2, 4},
		{"find first occurrence with cursor after first", []byte("hello world"), 6, 'o', 1, 4},
		{"find second occurrence with cursor after first", []byte("hello world"), 6, 'o', 2, -1},
		{"find unknown character", []byte("hello world"), 11, 'p', 1, -1},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			gb := NewGapBufferWithContent(test.input)
			result, err := gb.ReverseSeekToChar(test.cursor, test.char, test.count)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if result != test.expected {
				t.Errorf("expected %d, got %d", test.expected, result)
			}
		})
	}
}

func TestReverseSeekToChar_WithErrors(t *testing.T) {
	gb := NewGapBufferWithContent([]byte("hello world"))
	tests := []struct {
		name     string
		cursor   int
		char     byte
		count    int
		expected error
	}{
		{"negative cursor", -1, 'o', 1, errors.New("buffer: error converting cursor to buffer position when reverse seeking to character: cursor out of range: -1")},
		{"cursor greater than buffer content length", gb.Len() + 1, 'o', 1, errors.New("buffer: error converting cursor to buffer position when reverse seeking to character: cursor out of range: 12")},
		{"cursor greater than buffer length", len(gb.buffer) + 1, 'o', 1, errors.New("buffer: error converting cursor to buffer position when reverse seeking to character: cursor out of range: 32")},
		{"zero count", 0, 'o', 0, errors.New("buffer: expect positive count, got 0 instead")},
		{"negative count", 0, 'o', -1, errors.New("buffer: expect positive count, got -1 instead")},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result, err := gb.ReverseSeekToChar(test.cursor, test.char, test.count)
			if err == nil || err.Error() != test.expected.Error() {
				t.Errorf("expected error %v, got %v", test.expected, err)
			}
			if result != -1 {
				t.Errorf("expected -1 result, got %d", result)
			}
		})
	}
}

func TestRead_NoErrors(t *testing.T) {
	tests := []struct {
		name     string
		input    []byte
		cursor   int
		length   int
		expected []byte
	}{
		{"read from start to middle", []byte("hello world"), 0, 5, []byte("hello")},
		{"read from middle to start", []byte("hello world"), 6, 5, []byte("world")},
		{"read from start to end", []byte("hello world"), 0, 11, []byte("hello world")},
		{"read from start to past end", []byte("hello world"), 0, 20, []byte("hello world")},
		{"read from middle to past end", []byte("hello world"), 6, 20, []byte("world")},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			gb := NewGapBufferWithContent(test.input)
			result, err := gb.Read(test.cursor, test.length)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if string(result) != string(test.expected) {
				t.Errorf("expected %s, got %s", test.expected, result)
			}
		})
	}
}

func TestRead_WithErrors(t *testing.T) {
	gb := NewGapBufferWithContent([]byte("hello world"))
	tests := []struct {
		name     string
		cursor   int
		length   int
		expected error
	}{
		{"negative cursor", -1, 5, errors.New("buffer: error converting cursor to buffer position when reading: cursor out of range: -1")},
		{"cursor greater than buffer content length", gb.Len() + 1, 5, errors.New("buffer: error converting cursor to buffer position when reading: cursor out of range: 12")},
		{"cursor greater than buffer length", len(gb.buffer) + 1, 5, errors.New("buffer: error converting cursor to buffer position when reading: cursor out of range: 32")},
		{"negative length", 0, -1, errors.New("buffer: negative length -1 received")},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result, err := gb.Read(test.cursor, test.length)
			if err == nil || err.Error() != test.expected.Error() {
				t.Errorf("expected error %v, got %v", test.expected, err)
			}
			if result != nil {
				t.Errorf("expected nil result, got %s", result)
			}
		})
	}
}

func TestInsertByte_AppendOneByteEmptyBufferNoErrors(t *testing.T) {
	gb := NewGapBuffer()
	gb.InsertByte('a', 0)
	if gb.Len() != 1 {
		t.Errorf("expected length 1, got %d", gb.Len())
	}
	contents, err := gb.Read(0, 1)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if string(contents) != "a" {
		t.Errorf("expected 'a', got %s", contents)
	}
}
func TestInsertByte_AppendMultipleBytesEmptyBufferNoErrors(t *testing.T) {
	gb := NewGapBuffer()
	gb.InsertByte('a', 0)
	gb.InsertByte('b', 1)
	gb.InsertByte('c', 2)
	if gb.Len() != 3 {
		t.Errorf("expected length 3, got %d", gb.Len())
	}
	contents, err := gb.Read(0, 3)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	expected := "abc"
	if string(contents) != expected {
		t.Errorf("expected %s, got %s", expected, contents)
	}
}

func TestInsertByte_PrependMultipleBytesEmptyBufferNoErrors(t *testing.T) {
	gb := NewGapBuffer()
	gb.InsertByte('a', 0)
	gb.InsertByte('b', 0)
	gb.InsertByte('c', 0)
	if gb.Len() != 3 {
		t.Errorf("expected length 3, got %d", gb.Len())
	}
	contents, err := gb.Read(0, 3)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if string(contents) != "cba" {
		t.Errorf("expected 'a', got %s", contents)
	}
}

func TestInsertByte_AppendMultipleBytesNonEmptyBufferNoErrors(t *testing.T) {
	gb := NewGapBufferWithContent([]byte("xyz"))
	gb.InsertByte('a', 3)
	gb.InsertByte('b', 4)
	gb.InsertByte('c', 5)
	if gb.Len() != 6 {
		t.Errorf("expected length 6, got %d", gb.Len())
	}
	contents, err := gb.Read(0, 6)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	expected := "xyzabc"
	if string(contents) != expected {
		t.Errorf("expected %s, got %s", expected, contents)
	}
}

func TestInsertByte_PrependMultipleBytesNonEmptyBufferNoErrors(t *testing.T) {
	gb := NewGapBufferWithContent([]byte("xyz"))
	gb.InsertByte('a', 0)
	gb.InsertByte('b', 1)
	gb.InsertByte('c', 2)
	if gb.Len() != 6 {
		t.Errorf("expected length 6, got %d", gb.Len())
	}
	contents, err := gb.Read(0, 6)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	expected := "abcxyz"
	if string(contents) != expected {
		t.Errorf("expected %s, got %s", expected, contents)
	}
}

func TestInsertByte_ResizeBufferEmptyBufferNoErrors(t *testing.T) {
	gb := NewGapBuffer()
	expectedStr := ""
	for i := 0; i <= DEFAULT_BUFFER_SIZE; i++ {
		// Buffer should resize successfully
		// when buffer is full
		gb.InsertByte('a', 0)
		expectedStr += "a"
	}
	if gb.Len() != DEFAULT_BUFFER_SIZE+1 {
		t.Errorf("expected length %d, got %d", DEFAULT_BUFFER_SIZE+1, gb.Len())
	}
	contents, err := gb.Read(0, DEFAULT_BUFFER_SIZE+1)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if string(contents) != expectedStr {
		t.Errorf("expected %s, got %s", expectedStr, contents)
	}
}

func TestInsertByte_ResizeBufferNonEmptyBufferNoErrors(t *testing.T) {
	originalContents := make([]byte, DEFAULT_BUFFER_SIZE)
	for i := range DEFAULT_BUFFER_SIZE {
		originalContents[i] = 'a'
	}
	gb := NewGapBufferWithContent(originalContents)
	gb.InsertByte('b', DEFAULT_BUFFER_SIZE)
	if gb.Len() != DEFAULT_BUFFER_SIZE+1 {
		t.Errorf("expected length %d, got %d", DEFAULT_BUFFER_SIZE+1, gb.Len())
	}
	contents, err := gb.Read(0, DEFAULT_BUFFER_SIZE+1)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	expected := append(originalContents, 'b')
	if string(contents) != string(expected) {
		t.Errorf("expected %s, got %s", expected, contents)
	}
}

func TestInsertByte_EmptyBufferExpectErrors(t *testing.T) {
	gb := NewGapBuffer()
	tests := []struct {
		name          string
		inputPosition int
		expectedErr   error
	}{
		{"position less than 0", -1, errors.New("buffer: error converting cursor to buffer position when getting byte: cursor out of range: -1")},
		{"position greater than buffer size", 1, errors.New("buffer: error converting cursor to buffer position when getting byte: cursor out of range: 1")},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := gb.InsertByte('a', test.inputPosition)
			if err == nil || err.Error() != test.expectedErr.Error() {
				t.Errorf("expected error %v, got %v", test.expectedErr, err)
			}
		})
	}
}

func TestInsertByte_NonEmptyBufferExpectErrors(t *testing.T) {
	gb := NewGapBufferWithContent([]byte("xyz"))
	tests := []struct {
		name          string
		inputPosition int
		expectedErr   error
	}{
		{"position less than 0", -1, errors.New("buffer: error converting cursor to buffer position when getting byte: cursor out of range: -1")},
		{"position greater than buffer size", 4, errors.New("buffer: error converting cursor to buffer position when getting byte: cursor out of range: 4")},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := gb.InsertByte('a', test.inputPosition)
			if err == nil || err.Error() != test.expectedErr.Error() {
				t.Errorf("expected error %v, got %v", test.expectedErr, err)
			}
		})
	}
}

func TestDeleteByte_DeleteOneByteNoErrors(t *testing.T) {
	gb := NewGapBufferWithContent([]byte("xyz"))
	err := gb.DeleteByte(0)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if gb.Len() != 2 {
		t.Errorf("expected length 2, got %d", gb.Len())
	}
	contents, err := gb.Read(0, 2)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	expected := "yz"
	if string(contents) != expected {
		t.Errorf("expected %s, got %s", expected, contents)
	}
}

func TestDeleteByte_DeleteMultipleBytesNoErrors(t *testing.T) {
	gb := NewGapBufferWithContent([]byte("xyz"))
	for range 3 {
		err := gb.DeleteByte(0)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	}
	if gb.Len() != 0 {
		t.Errorf("expected length 2, got %d", gb.Len())
	}
	contents, err := gb.Read(0, 3)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	expected := ""
	if string(contents) != expected {
		t.Errorf("expected %s, got %s", expected, contents)
	}
}

func TestDeleteByte_EmptyBufferExpectError(t *testing.T) {
	gb := NewGapBuffer()
	err := gb.DeleteByte(0)
	expectedErr := errors.New("buffer: cannot delete byte from an empty buffer")
	if err == nil || err.Error() != expectedErr.Error() {
		t.Errorf("expected error %v, got %v", errors.New(""), err)
	}
}

func TestDeleteByte_NonEmptyBufferExpectError(t *testing.T) {
	gb := NewGapBufferWithContent([]byte("xyz"))
	tests := []struct {
		name          string
		inputPosition int
		expectedErr   error
	}{
		{"position less than 0", -1, errors.New("buffer: error converting cursor to buffer position when getting byte: cursor out of range: -1")},
		{"position greater than buffer size", 3, errors.New("buffer: cursor must be on a non-empty buffer position, got 3 instead")},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := gb.DeleteByte(test.inputPosition)
			if err == nil || err.Error() != test.expectedErr.Error() {
				t.Errorf("expected error %v, got %v", test.expectedErr, err)
			}
		})
	}
}

func TestDeleteByte_RepeatDeleteLastIndexExpectError(t *testing.T) {
	gb := NewGapBufferWithContent([]byte("xyz"))
	err := gb.DeleteByte(2)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if gb.Len() != 2 {
		t.Errorf("expected length 2, got %d", gb.Len())
	}
	contents, err := gb.Read(0, 2)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	expected := "xy"
	if string(contents) != expected {
		t.Errorf("expected %s, got %s", expected, contents)
	}
	err = gb.DeleteByte(2)
	expectedErr := errors.New("buffer: cursor must be on a non-empty buffer position, got 2 instead")
	if err == nil || err.Error() != expectedErr.Error() {
		t.Errorf("expected %v, got %v", expectedErr, err)
	}
}

func TestGetByte_ValidPositionNoErrors(t *testing.T) {
	gb := NewGapBufferWithContent([]byte("xyz"))
	b, err := gb.GetByte(0)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if b != 'x' {
		t.Errorf("expected 'x', got %c", b)
	}
	b, err = gb.GetByte(1)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if b != 'y' {
		t.Errorf("expected 'y', got %c", b)
	}
	b, err = gb.GetByte(2)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if b != 'z' {
		t.Errorf("expected 'z', got %c", b)
	}
}

func TestGetByte_EmptyBufferExpectError(t *testing.T) {
	gb := NewGapBuffer()
	b, err := gb.GetByte(0)
	expectedErr := errors.New("buffer: cursor out of range: 0")
	if err == nil || err.Error() != expectedErr.Error() {
		t.Errorf("expected error %v, got %v", expectedErr, err)
	}
	if b != 0 {
		t.Errorf("expected 0, got %d", b)
	}
}

func TestGetByte_NonEmptyBufferExpectError(t *testing.T) {
	gb := NewGapBufferWithContent([]byte("xyz"))
	tests := []struct {
		name          string
		inputPosition int
		expectedErr   error
	}{
		{"position less than 0", -1, errors.New("buffer: cursor out of range: -1")},
		{"position greater than buffer size", 3, errors.New("buffer: cursor out of range: 3")},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			b, err := gb.GetByte(test.inputPosition)
			if err == nil || err.Error() != test.expectedErr.Error() {
				t.Errorf("expected error %v, got %v", test.expectedErr, err)
			}
			if b != 0 {
				t.Errorf("expected 0, got %d", b)
			}
		})
	}
}

func TestLen_EmptyBufferNoErrors(t *testing.T) {
	gb := NewGapBuffer()
	if gb.Len() != 0 {
		t.Errorf("expected length 0, got %d", gb.Len())
	}
}

func TestLen_InsertIntoBufferNoErrors(t *testing.T) {
	gb := NewGapBufferWithContent([]byte("xyz"))
	if gb.Len() != 3 {
		t.Errorf("expected length 3, got %d", gb.Len())
	}
	gb.InsertByte('a', 0)
	if gb.Len() != 4 {
		t.Errorf("expected length 4, got %d", gb.Len())
	}
}

func TestLen_ResizeBufferNoErrors(t *testing.T) {
	gb := NewGapBuffer()
	for i := 0; i <= DEFAULT_BUFFER_SIZE; i++ {
		gb.InsertByte('a', 0)
	}
	if gb.Len() != DEFAULT_BUFFER_SIZE+1 {
		t.Errorf("expected length %d, got %d", DEFAULT_BUFFER_SIZE+1, gb.Len())
	}
}

func TestLen_DeleteFromBufferNoErrors(t *testing.T) {
	gb := NewGapBufferWithContent([]byte("xyz"))
	if gb.Len() != 3 {
		t.Errorf("expected length 3, got %d", gb.Len())
	}
	gb.DeleteByte(0)
	if gb.Len() != 2 {
		t.Errorf("expected length 2, got %d", gb.Len())
	}
	gb.DeleteByte(0)
	if gb.Len() != 1 {
		t.Errorf("expected length 1, got %d", gb.Len())
	}
	gb.DeleteByte(0)
	if gb.Len() != 0 {
		t.Errorf("expected length 0, got %d", gb.Len())
	}
}

func TestUndo_FreshBufferExpectError(t *testing.T) {
	gb := NewGapBuffer()
	_, err := gb.Undo()
	if err != nil {
		t.Errorf("expected no errors, got %v", err)
	}
}

func TestUndo_EmptyBufferUndoAfterInsertNoErrors(t *testing.T) {
	gb := NewGapBuffer()
	gb.InsertByte('a', 0)
	gb.InsertByte('b', 1)
	gb.InsertByte('c', 2)
	if gb.Len() != 3 {
		t.Errorf("expected length 3, got %d", gb.Len())
	}
	contents, err := gb.Read(0, 3)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	expected := "abc"
	if string(contents) != expected {
		t.Errorf("expected %s, got %s", expected, contents)
	}
	lastChange, err := gb.Undo()
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if lastChange == nil {
		t.Errorf("expected non-nil lastChange, got nil")
		return
	}
	if lastChange.Cursor != 0 {
		t.Errorf("expected cursor 0, got %d", lastChange.Cursor)
	}
	if lastChange.Length != 3 {
		t.Errorf("expected length 3, got %d", lastChange.Length)
	}
	if gb.Len() != 0 {
		t.Errorf("expected length 0, got %d", gb.Len())
	}
	contents, err = gb.Read(0, 3)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	expected = ""
	if string(contents) != expected {
		t.Errorf("expected %s, got %s", expected, contents)
	}
}

func TestUndo_NonEmptyBufferUndoAfterInsertNoErrors(t *testing.T) {
	gb := NewGapBufferWithContent([]byte("xyz"))
	gb.InsertByte('a', 0)
	gb.InsertByte('b', 1)
	gb.InsertByte('c', 2)
	if gb.Len() != 6 {
		t.Errorf("expected length 6, got %d", gb.Len())
	}
	contents, err := gb.Read(0, 6)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	expected := "abcxyz"
	if string(contents) != expected {
		t.Errorf("expected %s, got %s", expected, contents)
	}
	lastChange, err := gb.Undo()
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if lastChange == nil {
		t.Errorf("expected non-nil lastChange, got nil")
		return
	}
	if lastChange.Cursor != 0 {
		t.Errorf("expected cursor 0, got %d", lastChange.Cursor)
	}
	if lastChange.Length != 3 {
		t.Errorf("expected length 3, got %d", lastChange.Length)
	}
	if gb.Len() != 3 {
		t.Errorf("expected length 3, got %d", gb.Len())
	}
	contents, err = gb.Read(0, 3)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	expected = "xyz"
	if string(contents) != expected {
		t.Errorf("expected %s, got %s", expected, contents)
	}
}

func TestUndo_EmptyBufferUndoAfterInsertWithResizeNoErrors(t *testing.T) {
	gb := NewGapBuffer()
	expected := ""
	for i := range DEFAULT_BUFFER_SIZE + 1 {
		gb.InsertByte('a', i)
		expected += "a"
	}
	if gb.Len() != DEFAULT_BUFFER_SIZE+1 {
		t.Errorf("expected length %d, got %d", DEFAULT_BUFFER_SIZE+1, gb.Len())
	}
	contents, err := gb.Read(0, DEFAULT_BUFFER_SIZE+1)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if string(contents) != expected {
		t.Errorf("expected %s, got %s", expected, contents)
	}
	lastChange, err := gb.Undo()
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if lastChange == nil {
		t.Errorf("expected non-nil lastChange, got nil")
		return
	}
	if lastChange.Cursor != 0 {
		t.Errorf("expected cursor 0, got %d", lastChange.Cursor)
	}
	if lastChange.Length != DEFAULT_BUFFER_SIZE+1 {
		t.Errorf("expected length %d, got %d", DEFAULT_BUFFER_SIZE+1, lastChange.Length)
	}
	if gb.Len() != 0 {
		t.Errorf("expected length 0, got %d", gb.Len())
	}
}

func TestUndo_TwoUndoesAfterSequentialInsertWithWhitespace(t *testing.T) {
	gb := NewGapBufferWithContent([]byte("xyz"))
	gb.InsertByte('a', 3)
	gb.InsertByte('b', 4)
	gb.InsertByte('c', 5)
	gb.InsertByte(' ', 6)
	gb.InsertByte('1', 7)
	gb.InsertByte('2', 8)
	gb.InsertByte('3', 9)

	if gb.Len() != 10 {
		t.Errorf("expected length 10, got %d", gb.Len())
	}
	contents, err := gb.Read(0, 10)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	expected := "xyzabc 123"
	if string(contents) != expected {
		t.Errorf("expected %s, got %s", expected, contents)
	}
	lastChange, err := gb.Undo()
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if lastChange == nil {
		t.Errorf("expected non-nil lastChange, got nil")
		return
	}
	if lastChange.Cursor != 7 {
		t.Errorf("expected cursor 7, got %d", lastChange.Cursor)
	}
	if lastChange.Length != 3 {
		t.Errorf("expected length 3, got %d", lastChange.Length)
	}
	if gb.Len() != 7 {
		t.Errorf("expected length 7, got %d", gb.Len())
	}
	contents, err = gb.Read(0, 7)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	expected = "xyzabc "
	if string(contents) != expected {
		t.Errorf("expected %s, got %s", expected, contents)
	}
	lastChange, err = gb.Undo()
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if lastChange == nil {
		t.Errorf("expected non-nil lastChange, got nil")
		return
	}
	if lastChange.Cursor != 3 {
		t.Errorf("expected cursor 3, got %d", lastChange.Cursor)
	}
	if lastChange.Length != 4 {
		t.Errorf("expected length 4, got %d", lastChange.Length)
	}
	if gb.Len() != 3 {
		t.Errorf("expected length 3, got %d", gb.Len())
	}
	contents, err = gb.Read(0, 3)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	expected = "xyz"
	if string(contents) != expected {
		t.Errorf("expected %s, got %s", expected, contents)
	}
}

func TestUndo_TwoUndoesAfterAppendAndPrepend(t *testing.T) {
	gb := NewGapBufferWithContent([]byte("xyz"))
	gb.InsertByte('a', 3)
	gb.InsertByte('b', 4)
	gb.InsertByte('c', 5)
	gb.InsertByte('1', 0)
	gb.InsertByte('2', 1)
	gb.InsertByte('3', 2)

	if gb.Len() != 9 {
		t.Errorf("expected length 9, got %d", gb.Len())
	}
	contents, err := gb.Read(0, 9)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	expected := "123xyzabc"
	if string(contents) != expected {
		t.Errorf("expected %s, got %s", expected, contents)
	}
	lastChange, err := gb.Undo()
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if lastChange == nil {
		t.Errorf("expected non-nil lastChange, got nil")
		return
	}
	if lastChange.Cursor != 0 {
		t.Errorf("expected cursor 7, got %d", lastChange.Cursor)
	}
	if lastChange.Length != 3 {
		t.Errorf("expected length 3, got %d", lastChange.Length)
	}
	if gb.Len() != 6 {
		t.Errorf("expected length 7, got %d", gb.Len())
	}
	contents, err = gb.Read(0, 6)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	expected = "xyzabc"
	if string(contents) != expected {
		t.Errorf("expected %s, got %s", expected, contents)
	}
	lastChange, err = gb.Undo()
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if lastChange == nil {
		t.Errorf("expected non-nil lastChange, got nil")
		return
	}
	if lastChange.Cursor != 3 {
		t.Errorf("expected cursor 3, got %d", lastChange.Cursor)
	}
	if lastChange.Length != 3 {
		t.Errorf("expected length 3, got %d", lastChange.Length)
	}
	if gb.Len() != 3 {
		t.Errorf("expected length 3, got %d", gb.Len())
	}
	contents, err = gb.Read(0, 3)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	expected = "xyz"
	if string(contents) != expected {
		t.Errorf("expected %s, got %s", expected, contents)
	}
}

func TestUndo_NonEmptyBufferUndoAfterDelete(t *testing.T) {
	gb := NewGapBufferWithContent([]byte("xyz"))
	gb.DeleteByte(2)
	gb.DeleteByte(1)
	gb.DeleteByte(0)
	if gb.Len() != 0 {
		t.Errorf("expected length 0, got %d", gb.Len())
	}
	contents, err := gb.Read(0, 3)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	expected := ""
	if string(contents) != expected {
		t.Errorf("expected %s, got %s", expected, contents)
	}
	lastChange, err := gb.Undo()
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if lastChange == nil {
		t.Errorf("expected non-nil lastChange, got nil")
		return
	}
	if lastChange.Cursor != 2 {
		t.Errorf("expected cursor 2, got %d", lastChange.Cursor)
	}
	if lastChange.Length != 3 {
		t.Errorf("expected length 3, got %d", lastChange.Length)
	}
	expectedData := []byte("xyz")
	for i := range lastChange.Data {
		if lastChange.Data[len(lastChange.Data)-i-1] != expectedData[i] {
			t.Errorf("expected data %s, got %s", expectedData, lastChange.Data)
			break
		}
	}
	if gb.Len() != 3 {
		t.Errorf("expected length 3, got %d", gb.Len())
	}
}

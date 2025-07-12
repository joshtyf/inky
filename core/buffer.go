package core

import "os"

type Rune []byte

type Buffer interface {
	InsertRune(r rune, cursor int)
	SeekToChar(cursor int, char byte, count int) int
	ReverseSeekToChar(cursor int, char byte, count int) int
	Read(cursor int, length int) []byte
	Undo() *ChangeNode
	DeleteRune(cursor int)
	GetRune(cursor int) rune
	Len() int
	WriteTo(w *os.File) (int64, error)
}

type UndoType int

const (
	UndoInsert = iota
	UndoDelete
)

type ChangeNode struct {
	Cursor int
	Length int
	Data   []byte
}

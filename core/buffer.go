package core

type Rune []byte

type Buffer interface {
	InsertByte(b byte, cursor int)
	SeekToChar(cursor int, char byte, count int) int
	ReverseSeekToChar(cursor int, char byte, count int) int
	Read(cursor int, length int) []byte
	Undo() *ChangeNode
	DeleteByte(cursor int)
	GetByte(cursor int) byte
	Len() int
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

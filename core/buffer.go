package core

type Rune []byte

type Buffer interface {
	InsertByte(b byte, cursor int) error
	SeekToChar(cursor int, char byte, count int) (int, error)
	ReverseSeekToChar(cursor int, char byte, count int) (int, error)
	Read(cursor int, length int) ([]byte, error)
	Undo() (*ChangeNode, error)
	DeleteByte(cursor int) error
	GetByte(cursor int) (byte, error)
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

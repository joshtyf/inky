package main

type Rune []byte

type Buffer interface {
	InsertByte(b byte, cursor int) error
	FindNextLine(cursor int) (int, error)
	FindPrevLine(cursor int) (int, error)
	ReadTillNewLine(cursor int) (string, error)
	DeleteByte(cursor int) error
	GetByte(cursor int) (byte, error)
	Len() int
}

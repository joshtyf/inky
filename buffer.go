package main

type Rune []byte

type Buffer interface {
	InsertRune(r Rune, cursor int) error
	FindNextLine(cursor int) (int, error)
	FindPrevLine(cursor int) (int, error)
	ReadTillNewLine(cursor int) (string, error)
	Len() int
}

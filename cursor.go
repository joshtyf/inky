package main

type Cursor interface {
	MoveCursorRight(offset int)
	MoveCursorLeft(offset int)
	MoveCursorUp(offset int)
	MoveCursorDown(offset int)
}

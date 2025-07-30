package io

type ReadEditorLines func(start int, n int) ([][]byte, error)

type EditorState struct {
	KeyPressed    *Key
	CurrentLine   int
	CurrentColumn int
	EditorSaved   bool
	CharCount     int
	ReadEditorLines
}

type EditorIO interface {
	Start() (<-chan *Key, error)
	DisplayEditor(*EditorState) error
	Close() error
}

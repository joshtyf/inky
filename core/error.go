package core

type EditorClosedError struct{}

func (e *EditorClosedError) Error() string {
	return "editor closed"
}
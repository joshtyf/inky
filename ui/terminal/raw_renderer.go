package terminal

import (
	"github.com/joshtyf/inky/core"
)

type RawRenderer struct{}

func NewRawRenderer() *RawRenderer {
	return &RawRenderer{}
}

func (r *RawRenderer) Render(line int, es *core.EditorState) (string, error) {
	runes := es.GetLine(line)
	return string(runes), nil
}

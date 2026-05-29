package terminal

import (
	"github.com/joshtyf/inky/core"
)

type RawRenderer struct {
	lastRenderedVersion int
	renderCache         map[int]string
}

func NewRawRenderer() *RawRenderer {
	return &RawRenderer{
		lastRenderedVersion: -1,
		renderCache:         nil,
	}
}

func (r *RawRenderer) LastLine(es core.EditorState) int {
	return es.DocumentMaxLineLength - 1
}

func (r *RawRenderer) Render(line int, es core.EditorState) (string, error) {
	if r.renderCache == nil || es.Version != r.lastRenderedVersion {
		r.renderCache = make(map[int]string)
		r.lastRenderedVersion = es.Version
	}
	if cached, ok := r.renderCache[line]; ok {
		return cached, nil
	}
	r.renderCache[line] = string(es.GetLine(line))
	return r.renderCache[line], nil
}

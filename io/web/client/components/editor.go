package components

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/joshtyf/texteditor/io"
	"github.com/maxence-charriere/go-app/v10/pkg/app"
)

type editorControl struct {
	app.Compo
	handleSave app.EventHandler
}

func newEditorControl() *editorControl {
	return &editorControl{}
}
func (e *editorControl) Render() app.UI {
	return app.Div().Body(
		app.Button().
			Text("Save").
			Class("save-button").
			OnClick(e.handleSave),
	)
}

func (e *editorControl) OnSave(fn app.EventHandler) *editorControl {
	e.handleSave = fn
	return e
}

type editor struct {
	app.Compo
}

func (e *editor) Render() app.UI {
	return app.Div().Body(
		newEditorControl().OnSave(e.onSave),
		app.Textarea().Class("editor-container").OnKeyDown(e.onKeyDown),
	)
}

func (e *editor) onKeyDown(ctx app.Context, ev app.Event) {
	input := ev.Get("key").String()
	var k io.Key
	switch input {
	case "ArrowUp":
		k = io.Key{Code: io.ArrowUp}
	case "ArrowDown":
		k = io.Key{Code: io.ArrowDown}
	case "ArrowLeft":
		k = io.Key{Code: io.ArrowLeft}
	case "ArrowRight":
		k = io.Key{Code: io.ArrowRight}
	case "Backspace":
		k = io.Key{Code: io.Backspace}
	case "Enter":
		k = io.Key{Code: io.Newline}
	case "Escape":
		k = io.Key{Code: io.Escape}
	default:
		var r rune
		if len(input) == 1 {
			r = []rune(input)[0]
		} else {
			fmt.Println("Unrecognized key:", input)
			return
		}
		k = io.Key{Code: io.RuneKey, Rune: r}
	}

	body, err := json.Marshal(k)
	if err != nil {
		fmt.Println("Error marshaling key:", k)
		return
	}
	http.Post("/api/input", "application/json",
		bytes.NewBuffer(body),
	)
}

func (e *editor) onSave(ctx app.Context, ev app.Event) {
	// Handle the save event
	k := io.Key{Code: io.Save}
	body, err := json.Marshal(k)
	if err != nil {
		fmt.Println("Error marshaling key:", k)
		return
	}
	http.Post("/api/input", "application/json",
		bytes.NewBuffer(body),
	)
}

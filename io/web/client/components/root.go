package components

import "github.com/maxence-charriere/go-app/v10/pkg/app"

type Root struct {
	app.Compo
}

func (r *Root) Render() app.UI {
	return app.Div().Body(
		app.H1().Text("Welcome to the Web Text Editor!"),
		app.P().Text("This is a simple text editor built with Go and WebAssembly."),
		&editor{},
	)
}

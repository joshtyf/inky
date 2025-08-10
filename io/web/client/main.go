package main

import (
	"github.com/joshtyf/texteditor/io/web/client/components"
	"github.com/maxence-charriere/go-app/v10/pkg/app"
)

func main() {
	app.Route("/", func() app.Composer {
		return &components.Root{}
	})
	app.RunWhenOnBrowser()
}

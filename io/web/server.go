package web

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

	editorIO "github.com/joshtyf/texteditor/io"
	editorLog "github.com/joshtyf/texteditor/log"
	"github.com/maxence-charriere/go-app/v10/pkg/app"
	"github.com/spf13/viper"
)

func (io *EditorIO) startServer() {
	// Need to register the root route
	// to initialise the server
	// to be able to server the necessary files for our
	// Single Page app (e.g. wasm binary, app.js, etc.)
	// No components required. The wasm binary will be loaded
	// and the UI will be rendered by the client code
	app.Route("/", func() app.Composer {
		return &app.Compo{}
	})
	mux := http.NewServeMux()
	mux.Handle("/", &app.Handler{
		Title:       "Web Text Editor",
		Description: "A simple text editor built with Go and WebAssembly.",
	})
	mux.HandleFunc("/api/input", io.readClientKeyInput)
	io.server = &http.Server{
		Addr:    ":8080",
		Handler: mux,
	}
	err := io.server.ListenAndServe()
	if err != nil && err != http.ErrServerClosed {
		io.logger.Fatalf("error starting web server: %v", err)
		return
	}
}

type EditorIO struct {
	logger  *log.Logger
	inputCh chan *editorIO.Key
	server  *http.Server
}

func NewEditorIO(globalConf *viper.Viper) *EditorIO {
	logger := editorLog.CreateLogger("webIO")
	return &EditorIO{
		logger: logger,
	}
}

func (io *EditorIO) Start() (<-chan *editorIO.Key, error) {
	io.inputCh = make(chan *editorIO.Key)
	go func() {
		defer close(io.inputCh)
		io.startServer()
	}()
	return io.inputCh, nil
}

func (io *EditorIO) readClientKeyInput(w http.ResponseWriter, r *http.Request) {
	body := r.Body
	if body == nil {
		http.Error(w, "No input received", http.StatusBadRequest)
		return
	}
	defer body.Close()
	var key editorIO.Key
	if err := json.NewDecoder(body).Decode(&key); err != nil {
		http.Error(w, "Invalid input format", http.StatusBadRequest)
		return
	}
	io.inputCh <- &key
}

func (io *EditorIO) DisplayEditor(state *editorIO.EditorState) error {
	// This function would typically render the editor state to the web page.
	// For simplicity, we will just log the current line and column.
	io.logger.Printf("Current Line: %d, Current Column: %d", state.CurrentLine, state.CurrentColumn)
	return nil
}

func (io *EditorIO) Close() error {
	io.logger.Println("Closing web editor IO")
	io.logger.Println("Shutting down web server")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := io.server.Shutdown(ctx); err != nil {
		io.logger.Fatalf("Server forced to shutdown: %v", err)
	}
	io.logger.Println("Web server gracefully stopped")

	return nil
}

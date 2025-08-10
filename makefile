clean:
	rm -f web/app.wasm
	rm -f texteditor

build: clean
	GOARCH=wasm GOOS=js go build -o web/app.wasm ./io/web/client
	go build

run: build
	./texteditor editor_out.txt
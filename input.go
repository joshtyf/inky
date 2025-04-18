package main

import "context"

var defaultMapping = map[string]Key{
	// Arrow Keys
	"\x1b[A": {Code: ArrowUp},
	"\x1b[B": {Code: ArrowDown},
	"\x1b[D": {Code: ArrowLeft},
	"\x1b[C": {Code: ArrowRight},

	// Control Keys
	"\x04": {Code: CtrlD},
	"\x7f": {Code: Backspace},
	"\x1f": {Code: Undo},
	"\x0a": {Code: Newline},
}

var DEFAULT_INPUT = NewTerminalInput()

type Input interface {
	listen(keyOut chan<- Key)
}

type inputCtx struct {
	context.Context

	inputCh <-chan Key
}

func startInput(i Input) inputCtx {
	inputCh := make(chan Key)
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		i.listen(inputCh)
		cancel()
	}()

	return inputCtx{
		Context: ctx,
		inputCh: inputCh,
	}
}

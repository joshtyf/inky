package main

import (
	"context"
	"log"
)

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

type Input interface {
	start() error
	listen() (*Key, error)
	stop() error
}

type inputCtx struct {
	context.Context

	inputCh <-chan *Key
}

func startAndListen(i Input) (ret inputCtx) {
	inputCh := make(chan *Key)
	ctx, cancel := context.WithCancelCause(context.Background())
	ret = inputCtx{
		Context: ctx,
		inputCh: inputCh,
	}

	err := i.start()
	if err != nil {
		log.Printf("error starting input: %v", err)
		cancel(err)
	}

	go func() {
		for {
			k, err := i.listen()
			if err != nil || k.Code == CtrlD {
				cancel(err)
				close(inputCh)
				return
			} else {
				inputCh <- k
			}
		}

	}()
	return
}

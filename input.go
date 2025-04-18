package main

import "context"

type SpecialKey int

const (
	ArrowUp SpecialKey = iota
	ArrowDown
	ArrowLeft
	ArrowRight
	Delete
	NewLine
	Undo
)

var DEFAULT_INPUT = NewTerminalInput()

type Input interface {
	transmit(charOut chan<- byte, keyOut chan<- SpecialKey)
}

type inputCtx struct {
	context.Context

	charOut <-chan byte
	keyOut  <-chan SpecialKey
}

func startInput(i Input) inputCtx {
	charOut := make(chan byte)
	keyOut := make(chan SpecialKey)
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		i.transmit(charOut, keyOut)
		cancel()
	}()

	return inputCtx{
		Context: ctx,
		charOut: charOut,
		keyOut:  keyOut,
	}
}

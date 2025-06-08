package core

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

type input interface {
	start() (<-chan *Key, error)
	close() error
}

package terminal

import "fmt"

// Text attribute sequences.
const (
	AnsiReset   = "\x1b[0m"
	AnsiBold    = "\x1b[1m"
	AnsiDim     = "\x1b[2m"
	AnsiItalic  = "\x1b[3m"
	AnsiInverse = "\x1b[7m"
)

// Cursor and screen control sequences.
const (
	AnsiClearScreen   = "\x1b[2J"
	AnsiCursorHome    = "\x1b[H"
	AnsiCursorShow    = "\x1b[?25h"
	AnsiCursorSave    = "\x1b[s"
	AnsiCursorRestore = "\x1b[u"
	AnsiEraseLine     = "\x1b[2K"
)

// AnsiMoveCursor returns the escape sequence to move the cursor to the given
// 1-based row and column.
func AnsiMoveCursor(row, col int) string {
	return fmt.Sprintf("\x1b[%d;%dH", row, col)
}

// AnsiMoveToLineStart returns the escape sequence to move the cursor to the
// start (column 1) of the given 1-based row.
func AnsiMoveToLineStart(row int) string {
	return fmt.Sprintf("\x1b[%d;1H", row)
}

// AnsiHyperlink wraps label in an OSC 8 hyperlink pointing to url.
func AnsiHyperlink(url, label string) string {
	return "\x1b]8;;" + url + "\x1b\\" + label + "\x1b]8;;\x1b\\"
}

package main

import "github.com/gammazero/deque"

type Command interface {
	execute() error
	undo() error
}

type NoOpCommand struct{}

func (n *NoOpCommand) execute() error {
	return nil
}
func (n *NoOpCommand) undo() error {
	return nil
}

type MoveCursorCommand struct {
	cm        *CursorMgr
	direction SpecialKey
}

func (m *MoveCursorCommand) execute() error {
	switch m.direction {
	case ArrowUp:
		m.cm.MoveCursorUp()
	case ArrowDown:
		m.cm.MoveCursorDown()
	case ArrowLeft:
		m.cm.MoveCursorLeft()
	case ArrowRight:
		m.cm.MoveCursorRight()
	default:
		return nil
	}
	return nil
}

func (m *MoveCursorCommand) undo() error {
	switch m.direction {
	case ArrowUp:
		m.cm.MoveCursorDown()
	case ArrowDown:
		m.cm.MoveCursorUp()
	case ArrowLeft:
		m.cm.MoveCursorRight()
	case ArrowRight:
		m.cm.MoveCursorLeft()
	default:
		return nil
	}
	return nil
}

type InsertCharCommand struct {
	cm  *CursorMgr
	buf Buffer
	c   byte
}

func (i *InsertCharCommand) execute() error {
	i.cm.InsertAtCursor(i.c, i.buf)
	return nil
}

func (i *InsertCharCommand) undo() error {
	i.cm.BackspaceAtCursor(i.buf)
	return nil
}

type BackspaceCommand struct {
	cm      *CursorMgr
	buf     Buffer
	deleted byte
}

func (b *BackspaceCommand) execute() error {
	b.deleted = b.cm.BackspaceAtCursor(b.buf)
	return nil
}

func (b *BackspaceCommand) undo() error {
	b.cm.InsertAtCursor(b.deleted, b.buf)
	return nil
}

const HISTORY_SIZE = 20

type CommandController struct {
	cm      *CursorMgr
	buf     Buffer
	history *deque.Deque[Command]
}

func NewCommandController(cm *CursorMgr, buf Buffer) *CommandController {
	history := &deque.Deque[Command]{}
	history.Grow(HISTORY_SIZE)
	return &CommandController{
		cm:      cm,
		buf:     buf,
		history: history,
	}
}

func (cc *CommandController) processInput(charIn <-chan byte, keyIn <-chan SpecialKey) <-chan Command {
	commandCh := make(chan Command)
	go func() {
		for {
			var cmd Command
			select {
			case input := <-charIn:
				cmd = &InsertCharCommand{cc.cm, cc.buf, input}
			case input := <-keyIn:
				switch input {
				case ArrowUp, ArrowDown, ArrowLeft, ArrowRight:
					cmd = &MoveCursorCommand{cc.cm, input}
				case Delete:
					cmd = &BackspaceCommand{cc.cm, cc.buf, 0}
				case NewLine:
					cmd = &InsertCharCommand{cc.cm, cc.buf, '\n'}
				case Undo:
					// Naive undo implementation
					// TODO: implement a better undo logic e.g. undo entire word
					if cc.history.Len() > 0 {
						lastCommand := cc.history.PopFront()
						if lastCommand != nil {
							lastCommand.undo()
						}
					}
					cmd = &NoOpCommand{}
				}
			}
			if cmd != nil {
				commandCh <- cmd
				switch cmd.(type) {
				case *NoOpCommand:
					// Don't add NoOpCommand to history
				default:
					if cc.history.Len() == HISTORY_SIZE {
						cc.history.PopBack()
					}
					cc.history.PushFront(cmd)
				}
			}
		}
	}()
	return commandCh
}

package terminal

import (
	"fmt"
	"log"
	"os"
	"strings"

	editorIO "github.com/joshtyf/texteditor/io"
	editorLog "github.com/joshtyf/texteditor/log"
	"github.com/spf13/viper"
	"golang.org/x/term"
)

func setDefaultConfig(conf *viper.Viper) {
	if conf == nil {
		conf = viper.New()
	}
	conf.SetDefault("showCharCount", true)
	// TODO: create key mapping for terminal input
}

type EditorIO struct {
	top           int
	logger        *log.Logger
	input         *input
	output        *output
	showCharCount bool
}

func NewEditorIO(globalConf *viper.Viper) *EditorIO {
	logger := editorLog.CreateLogger("terminalIO")
	conf := globalConf.Sub("terminal")
	setDefaultConfig(conf)
	return &EditorIO{
		top:           0,
		logger:        logger,
		input:         newInput(logger, nil),
		output:        newOutput(logger),
		showCharCount: conf.GetBool("showCharCount"),
	}
}

func (io *EditorIO) Start() (<-chan *editorIO.Key, error) {
	err := io.setup()
	if err != nil {
		return nil, fmt.Errorf("error initialising editor io: %w", err)
	}
	inputCh := make(chan *editorIO.Key)
	go func() {
		defer close(inputCh)

		for {
			keys, err := io.input.read()
			if err != nil {
				panic(fmt.Sprintf("error reading input: %v", err))
			}
			for _, k := range keys {
				inputCh <- k
			}
		}
	}()
	return inputCh, nil
}

func (io *EditorIO) DisplayEditor(es *editorIO.EditorState) error {
	_, h, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil {
		return fmt.Errorf("error getting terminal size for display: %w", err)
	}
	io.output.clearScreen()
	// Reposition screen to match editor view
	if es.CurrentLine < io.top {
		io.top = es.CurrentLine
	} else if es.CurrentLine >= io.top+h-1 {
		io.top = es.CurrentLine - h + 2
	}
	content, err := es.ReadEditorLines(io.top, h-1) // Last line reserved for status line
	if err != nil {
		return fmt.Errorf("error reading editor lines: %w", err)
	}
	for i := range content {
		io.output.writeLine(i, content[i])
	}
	io.writeStatusLine(h-1, es)
	io.output.moveCursor(es.CurrentLine-io.top, es.CurrentColumn)
	return nil
}

func (io *EditorIO) writeStatusLine(line int, es *editorIO.EditorState) {
	statusConf := []string{}
	if io.showCharCount {
		statusConf = append(statusConf, fmt.Sprintf("Char Count %d", es.CharCount))
	}
	statusConf = append(statusConf, fmt.Sprintf("Line %d, Column %d", es.CurrentLine+1, es.CurrentColumn+1))
	statusConf = append(statusConf, fmt.Sprintf("Saved: %t", es.EditorSaved))
	status := strings.Join(statusConf, " | ")

	io.output.writeRawLine(line, status)
}

func (io *EditorIO) setup() error {
	if err := io.input.setup(); err != nil {
		return fmt.Errorf("error setting up terminal input: %w", err)
	}
	if err := io.output.setup(); err != nil {
		return fmt.Errorf("error setting up terminal output: %w", err)
	}
	return nil
}

func (io *EditorIO) Close() error {
	if err := io.input.reset(); err != nil {
		return fmt.Errorf("error resetting terminal input: %w", err)
	}
	if err := io.output.reset(); err != nil {
		return fmt.Errorf("error resetting terminal output: %w", err)
	}
	return nil
}

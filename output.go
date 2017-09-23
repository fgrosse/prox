package prox

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"sync"
)

type output struct {
	mu     sync.Mutex
	writer io.Writer
	colors *colorPalette
}

func newOutput() *output {
	return &output{
		writer: os.Stdout,
		colors: newColorPalette(),
	}
}

func (o *output) next(name string, longestName int) *processOutput {
	return &processOutput{
		Writer: o,
		name:   name,
		format: fmt.Sprintf("%%s%%-%d.%ds │%%s %%s", longestName, longestName),
		color:  o.colors.next(),
	}
}

// Write implements io.Writer by delegating all writes to os writer in a
// synchronized manner.
func (o *output) Write(b []byte) (int, error) {
	o.mu.Lock()
	defer o.mu.Unlock()

	return o.writer.Write(b)
}

type processOutput struct {
	io.Writer
	name   string
	format string
	color  color
}

func (o *processOutput) Write(p []byte) (int, error) {
	msg := o.formatMsg(p)
	_, err := fmt.Fprintln(o.Writer, msg)
	return len(p), err
}

func (o *processOutput) formatMsg(p []byte) string {
	msg := new(bytes.Buffer)
	for _, line := range bytes.Split(bytes.TrimSpace(p), []byte("\n")) {
		if msg.Len() > 0 {
			msg.WriteString("\n")
		}
		fmt.Fprintf(msg, o.format,
			colorBold+o.color,
			o.name,
			colorDefault,
			line,
		)
	}

	return msg.String()
}

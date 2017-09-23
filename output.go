package prox

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
)

// output provides synchronized and colored *processOutput instances.
type output struct {
	mu           sync.Mutex
	writer       io.Writer
	colors       *colorPalette
	prefixLength int
}

func newOutput(pp []Process) *output {
	return &output{
		writer:       os.Stdout,
		colors:       newColorPalette(),
		prefixLength: longestName(pp),
	}
}

func longestName(pp []Process) int {
	var longest string
	for _, p := range pp {
		if n := p.Name(); len(n) > len(longest) {
			longest = n
		}
	}

	n := len(longest)
	if n < 8 {
		n = 8
	}

	return n
}

// next creates a new *processOutput using the next color of the color palette.
func (o *output) next(name string) *processOutput {
	c := o.colors.next()
	return o.nextColored(name, c)
}

func (o *output) nextColored(name string, c color) *processOutput {
	name += strings.Repeat(" ", o.prefixLength-len(name))
	return &processOutput{
		Writer: o,
		prefix: fmt.Sprint(colorDefault, colorBold, c, name, " │ ", colorDefault),
	}
}

// Write implements io.Writer by delegating all writes to os writer in a
// synchronized manner.
func (o *output) Write(b []byte) (int, error) {
	o.mu.Lock()
	defer o.mu.Unlock()

	return o.writer.Write(b)
}

// processOutput is an io.Writer that is used to write all output of a single
// process. New processOutput instances should be created via output.next(…).
type processOutput struct {
	io.Writer
	prefix string
}

// Write implements io.writer by formatting b and writing it through os wrapped
// io.Writer.
func (o *processOutput) Write(b []byte) (int, error) {
	msg := o.formatMsg(b)
	_, err := fmt.Fprintln(o.Writer, msg)
	return len(b), err
}

func (o *processOutput) formatMsg(p []byte) string {
	msg := new(bytes.Buffer)
	for _, line := range bytes.Split(bytes.TrimSpace(p), []byte("\n")) {
		if msg.Len() > 0 {
			msg.WriteString("\n")
		}
		fmt.Fprint(msg, o.prefix, string(line))
	}

	return msg.String()
}

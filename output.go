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

func newOutput(pp []Process, noColors bool) *output {
	o := &output{
		writer:       os.Stdout,
		prefixLength: longestName(pp, 8),
	}

	if !noColors {
		o.colors = newColorPalette()
	}

	return o
}

func longestName(pp []Process, minLength int) int {
	var longest string
	for _, p := range pp {
		if n := p.Name(); len(n) > len(longest) {
			longest = n
		}
	}

	n := len(longest)
	if n < minLength {
		n = minLength
	}

	return n
}

// next creates a new *processOutput using the next color of the color palette.
func (o *output) next(name string) *processOutput {
	c := o.colors.next()
	return o.nextColored(name, c)
}

// nextColored creates a new *processOutput using the provided color.
func (o *output) nextColored(name string, c color) *processOutput {
	name += strings.Repeat(" ", o.prefixLength-len(name))
	po := newProcessOutput(o)
	if c == colorNone {
		po.prefix = name + " │ "
	} else {
		po.prefix = fmt.Sprint(colorDefault, colorBold, c, name, " │ ", colorDefault)
	}

	return po
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
	mu      sync.Mutex
	writers []io.Writer
	prefix  string
}

func newProcessOutput(w io.Writer) *processOutput {
	return &processOutput{
		writers: []io.Writer{w},
	}
}

// AddWriter adds a new writer that will receive all messages that are written
// via o.
func (o *processOutput) AddWriter(w io.Writer) {
	o.mu.Lock()
	o.writers = append(o.writers, w)
	o.mu.Unlock()
}

// RemoveWriter removes a previously added writer from the output.
func (o *processOutput) RemoveWriter(w io.Writer) {
	o.mu.Lock()
	ww := make([]io.Writer, 0, len(o.writers))
	for _, x := range o.writers {
		if x != w {
			ww = append(ww, x)
		}
	}
	o.writers = ww
	o.mu.Unlock()
}

// Write implements io.writer by formatting b and writing it through os wrapped
// io.Writer.
func (o *processOutput) Write(b []byte) (int, error) {
	o.mu.Lock()
	w := io.MultiWriter(o.writers...)
	o.mu.Unlock()

	msg := o.formatMsg(b)
	_, err := fmt.Fprintln(w, msg)
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

package prox

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"time"
)

type output struct {
	io.Writer
	colors *colorProvider
}

func newOutput() *output {
	return &output{
		Writer: os.Stdout,
		colors: newColorProvider(),
	}
}

func (o *output) next(name string, longestName int) *processOutput {
	return &processOutput{
		Writer: o.Writer,
		name:   name,
		format: fmt.Sprintf("%%s %%-%d.%ds %%s", longestName, longestName),
		color:  o.colors.next(),
	}
}

type processOutput struct {
	io.Writer
	name   string
	format string
	color  color
}

// TODO: test lots of concurrent output from multiple processes
func (o *processOutput) Write(p []byte) (int, error) {
	msg := o.formatMsg(p)
	_, err := fmt.Fprintln(o.Writer, msg) // TODO: synchronize writing
	return len(p), err
}

func (o *processOutput) formatMsg(p []byte) string {
	return o.color.apply(fmt.Sprintf(o.format,
		time.Now().Format("15:04:05"),
		o.name,
		bytes.TrimSpace(p),
	))
}

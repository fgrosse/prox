package prox

import (
	"bytes"
	"fmt"
	"io"
	"os"
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
		format: fmt.Sprintf("%%s %%-%d.%ds │%%s %%s", longestName, longestName),
		color:  o.colors.next(),
	}
}

type processOutput struct {
	io.Writer
	name   string
	format string
	color  color
}

func (o *processOutput) Write(p []byte) (int, error) {
	msg := o.formatMsg(p)
	_, err := fmt.Fprintln(o.Writer, msg) // TODO: synchronize writing
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

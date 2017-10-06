package prox

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"sync"

	"github.com/pkg/errors"
)

// output provides synchronized and colored *processOutput instances.
type output struct {
	mu           sync.Mutex
	writer       io.Writer
	colors       *colorPalette
	prefixLength int
}

// processOutput is an io.Writer that is used to write all output of a single
// process. New processOutput instances should be created via output.next(…).
type processOutput struct {
	mu      sync.Mutex
	writers []io.Writer
	prefix  string
}

func newOutput(pp []Process, noColors bool, w io.Writer) *output {
	o := &output{
		writer:       w,
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

// Write implements io.Writer by delegating all writes to o.writer in a
// synchronized manner.
func (o *output) Write(b []byte) (int, error) {
	o.mu.Lock()
	defer o.mu.Unlock()

	return o.writer.Write(b)
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

type processJSONOutput struct {
	writer io.Writer
	buffer *bytes.Buffer
	reader *bufio.Reader

	messageField string
	levelField   string
}

func newProcessJSONOutput(w io.Writer) *processJSONOutput {
	b := new(bytes.Buffer)
	return &processJSONOutput{
		writer:       w,
		buffer:       b,
		reader:       bufio.NewReader(b),
		messageField: "message",
		levelField:   "level",
	}
}

func (o *processJSONOutput) Write(p []byte) (int, error) {
	n, err := o.buffer.Write(p)
	if err != nil {
		return n, err
	}

	line, err := o.reader.ReadBytes('\n')
	if err == io.EOF {
		// we did not write enough data into the buffer yet
		return n, nil
	}
	if err != nil {
		return n, errors.Wrap(err, "line buffer")
	}

	// TODO: check the read parts are eventually freed from the buffer
	m := map[string]interface{}{}
	err = json.Unmarshal(line, &m)
	if err != nil {
		return n, errors.Wrap(err, "parsing JSON message")
	}

	msg := o.stringField(m, o.messageField)
	delete(m, o.messageField)

	var col color
	if lvl := o.stringField(m, o.levelField); lvl != "" {
		delete(m, o.levelField)
		if msg != "" {
			msg = "\t" + msg
		}
		msg = fmt.Sprintf("[%s]%s", strings.ToUpper(lvl), msg)

		if lvl == "error" {
			col = colorRed
		}
	}

	if len(m) > 0 {
		extra, err := o.prettyJSON(m)
		if err != nil {
			return n, err
		}
		msg = msg + "\t" + extra
	}

	if col != "" {
		msg = colored(col, msg)
	}

	_, err = o.writer.Write([]byte(msg + "\n"))
	return n, err
}

func (*processJSONOutput) stringField(m map[string]interface{}, key string) string {
	v, ok := m[key]
	if !ok {
		return ""
	}

	s, ok := v.(string)
	if !ok {
		return ""
	}

	return s
}

func (*processJSONOutput) prettyJSON(i interface{}) (string, error) {
	b, err := json.MarshalIndent(i, "", "")
	if err != nil {
		return "", err
	}

	b = bytes.Map(func(r rune) rune {
		if r == '\n' {
			return ' '
		}
		return r
	}, b)

	return string(b), nil
}

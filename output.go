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

// output provides synchronized and colored *formattedOutput instances.
// Every Executor should always use a single *output instance to create the
// io.Writers for the processes it spawns.
type output struct {
	writer       *syncWriter
	colors       *colorPalette
	prefixLength int
}

// syncWriter decorates an io.Writer (i.e. a *output) with synchronization.
type syncWriter struct {
	sync.Mutex
	io.Writer
}

func newSyncWriter(w io.Writer) *syncWriter {
	return &syncWriter{Writer: w}
}

// Write implements io.Writer by delegating all writes to o.writer in a
// synchronized manner.
func (w *syncWriter) Write(b []byte) (int, error) {
	w.Lock()
	defer w.Unlock()

	return w.Writer.Write(b)
}

func newOutput(pp []Process, noColors bool, w io.Writer) *output {
	o := &output{
		writer:       newSyncWriter(w),
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
		if n := p.Name; len(n) > len(longest) {
			longest = n
		}
	}

	n := len(longest)
	if n < minLength {
		n = minLength
	}

	return n
}

// next creates a writer that can be used as output of a process using the next
// color of the color palette. If the process is configured to emit JSON log
// messages the writer will decode them in order to provide extended
// functionality. Additionally the output will have a colored prefix in order to
// display the outputs of multiple processes side by side in a single shell.
func (o *output) next(p Process) *multiWriter {
	c := o.colors.next()
	return o.nextColored(p, c)
}

// nextColored is like output.next(…) but allows to set the color directly.
func (o *output) nextColored(p Process, c color) *multiWriter {
	po := &formattedOutput{Writer: o.writer}
	name := p.Name + strings.Repeat(" ", o.prefixLength-len(p.Name))
	if c == colorNone {
		po.prefix = name + " │ "
	} else {
		po.prefix = fmt.Sprint(colorDefault, colorBold, c, name, " │ ", colorDefault)
	}

	var w io.Writer = po
	if p.JSONOutput {
		w = newProcessJSONOutput(po)
	}

	return newMultiWriter(w)
}

type multiWriter struct {
	mu      sync.Mutex
	writers []io.Writer
}

func newMultiWriter(w io.Writer) *multiWriter {
	return &multiWriter{writers: []io.Writer{w}}
}

// AddWriter adds a new writer that will receive all messages that are written
// via o.
func (mw *multiWriter) AddWriter(w io.Writer) {
	mw.mu.Lock()
	mw.writers = append(mw.writers, w)
	mw.mu.Unlock()
}

// RemoveWriter removes a previously added writer from the output.
func (mw *multiWriter) RemoveWriter(w io.Writer) {
	mw.mu.Lock()
	ww := make([]io.Writer, 0, len(mw.writers))
	for _, x := range mw.writers {
		if x != w {
			ww = append(ww, x)
		}
	}
	mw.writers = ww
	mw.mu.Unlock()
}

// Write implements io.writer by writing p via all its writers. This function
// returns without an error if at least one of the writers has written the
// message without an error.
func (mw *multiWriter) Write(p []byte) (int, error) {
	var lastErr error
	var ok bool

	mw.mu.Lock()
	for _, w := range mw.writers {
		n, err := w.Write(p)
		if err != nil {
			lastErr = err
			continue
		}
		if n != len(p) {
			lastErr = io.ErrShortWrite
			continue
		}

		ok = true
	}
	mw.mu.Unlock()

	if !ok {
		return 0, lastErr
	}

	// at least one writer has successfully written the message
	return len(p), nil
}

// formattedOutput is an io.Writer that is used to write all output of a single
// process. New formattedOutput instances should be created via output.next(…).
type formattedOutput struct {
	io.Writer
	prefix string
}

// Write implements io.writer by formatting b and writing it through os wrapped
// io.Writer.
func (o *formattedOutput) Write(b []byte) (int, error) {
	msg := o.formatMsg(b)
	_, err := fmt.Fprintln(o.Writer, msg)
	return len(b), err
}

func (o *formattedOutput) formatMsg(p []byte) string {
	msg := new(bytes.Buffer)
	for _, line := range bytes.Split(bytes.TrimSpace(p), []byte("\n")) {
		if msg.Len() > 0 {
			msg.WriteString("\n")
		}
		fmt.Fprint(msg, o.prefix, string(line))
	}

	return msg.String()
}

// a bufferedWriter is an io.Writer that buffers written messages until the next
// new line character and then write every line via its embedded writer.
type bufferedWriter struct {
	io.Writer               // the writer we are eventually emitting our output to
	buffer    *bytes.Buffer // contains all bytes written up to the next new line
	reader    *bufio.Reader // used to read lines from the buffer
}

func newBufferedProcessOutput(w io.Writer) io.Writer {
	b := new(bytes.Buffer)
	return &bufferedWriter{
		Writer: w,
		buffer: b,
		reader: bufio.NewReader(b),
	}
}

func (o *bufferedWriter) Write(p []byte) (int, error) {
	n, err := o.buffer.Write(p)
	if err != nil {
		return n, err
	}

	// TODO: test writing multiple lines in a single call (needs to loop until io.EOF)
	line, err := o.reader.ReadBytes('\n')
	if err == io.EOF {
		// we did not write enough data into the buffer yet so we return and
		// wait for more data to be written in the future.
		return n, nil
	}
	if err != nil {
		return n, errors.Wrap(err, "line buffer")
	}

	// TODO: check that the read parts are eventually freed from the buffer
	_, err = o.Writer.Write(line)
	return n, err
}

// a processJSONOutput is an io.Writer for processes which emit structured JSON
// messages. This writer expects that it will always receive complete json
// encoded messages on each write. Thus it is usually best to wrap each
// processJSONOutput into a bufferedWriter.
type processJSONOutput struct {
	io.Writer    // the writer we are eventually emitting our formatted output to
	messageField string
	levelField   string

	// taggingRules is an ordered list of functions that tag a structured log message
	taggingRules []func(map[string]interface{}) (tag string)

	// tagActions maps tags to the action that should be applied to the tagged message
	tagActions map[string]tagAction
}

type tagAction struct {
	color color
}

func newProcessJSONOutput(w io.Writer) io.Writer {
	return newBufferedProcessOutput(&processJSONOutput{
		Writer:       w,
		messageField: "message",
		levelField:   "level",
	})
}

// addTaggingRule adds a new tagging rule to o. The `tag` is applied to each
// message which contains a certain `field` where the corresponding value is
// equal to the given `value`.
func (o *processJSONOutput) addTaggingRule(field, value, tag string) {
	o.taggingRules = append(o.taggingRules, func(m map[string]interface{}) string {
		if o.stringField(m, field) != value {
			return ""
		}

		return tag
	})
}

// setTagAction instructs o to perform a certain action to all messages that
// have been tagged with `tag` (e.g. change the log color).
func (o *processJSONOutput) setTagAction(tag string, action tagAction) {
	if o.tagActions == nil {
		o.tagActions = map[string]tagAction{}
	}

	o.tagActions[tag] = action
}

func (o *processJSONOutput) Write(line []byte) (int, error) {
	m := map[string]interface{}{}
	err := json.Unmarshal(line, &m)
	if err != nil {
		return 0, errors.Wrap(err, "parsing JSON message")
	}

	var col color
	tags := o.applyTags(m)
	for _, t := range tags {
		action, ok := o.tagActions[t]
		if !ok {
			continue
		}

		if action.color != "" {
			col = action.color
		}
	}

	msg := o.stringField(m, o.messageField)
	lvl := o.stringField(m, o.levelField)
	delete(m, o.messageField)
	delete(m, o.levelField)

	if lvl != "" {
		msg = fmt.Sprintf("[%s]\t%s", strings.ToUpper(lvl), msg)
	}

	if len(m) > 0 {
		extra, err := o.prettyJSON(m)
		if err != nil {
			return 0, err
		}
		msg = msg + "\t" + extra
	}

	if col != "" {
		msg = colored(col, msg)
	}

	return o.Writer.Write([]byte(msg + "\n"))
}

// stringField attempts to extract a string field stored under the given key in
// the map. The empty string is returned if no such key exists in m or if its
// value is not a string.
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

func (o *processJSONOutput) applyTags(m map[string]interface{}) []string {
	var tags []string
	for _, f := range o.taggingRules {
		if tag := f(m); tag != "" {
			tags = append(tags, tag)
		}
	}
	return tags
}

// prettyJSON marshals i into a JSON pretty printed single line format.
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

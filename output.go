package prox

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"regexp"
	"strings"
	"sync"

	"github.com/pkg/errors"
)

// DefaultStructuredOutput returns the default configuration for processes that
// do not specify structured log output specifically.
func DefaultStructuredOutput(env Environment) StructuredOutput {
	msgField := env.Get("PROX_MESSAGE_FIELD", "msg")
	lvlField := env.Get("PROX_LEVEL_FIELD", "level")

	return StructuredOutput{
		Format:       "auto",
		MessageField: msgField,
		LevelField:   lvlField,
		TagColors: map[string]string{
			"error": "red-bold",
			"fatal": "red-bold",
		},
		TaggingRules: []TaggingRule{
			{
				Tag:   "error",
				Field: lvlField,
				Value: "/(ERR(O|OR)?)|(WARN(ING)?)/i",
			},
			{
				Tag:   "fatal",
				Field: lvlField,
				Value: "/FATAL?|PANIC/i",
			},
		},
	}
}

// StructuredOutput contains all configuration to setup advanced functionality
// for structured logs.
type StructuredOutput struct {
	Format       string // e.g. "json", the zero value makes prox auto-detect the format
	MessageField string
	LevelField   string

	TaggingRules []TaggingRule
	TagColors    map[string]string
}

// A TaggingRule may be applied to a structured log message to tag it. These
// tags can then later be used to change the log lines appearance or to modify
// its content.
type TaggingRule struct {
	Field string
	Value string // either a concrete string or a regex like so: "/error|fatal/i"
	Tag   string
}

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
	out := &formattedOutput{Writer: o.writer}
	name := p.Name + strings.Repeat(" ", o.prefixLength-len(p.Name))
	if c == colorNone {
		out.prefix = name + " │ "
	} else {
		out.prefix = fmt.Sprint(colorDefault, colorBold, c, name, " │ ", colorDefault)
	}

	if p.Output.Format == "" {
		p.Output = DefaultStructuredOutput(p.Env)
	}

	var w io.Writer = out
	switch p.Output.Format {
	case "json":
		jo := newProcessJSONOutput(out, p.Output)
		w = newBufferedProcessOutput(jo)
	default:
		ao := newProcessAutoDetectOutput(out, p.Output)
		w = newBufferedProcessOutput(ao)
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
// new line character and then writes every line via its embedded writer.
type bufferedWriter struct {
	io.Writer               // the writer we are eventually emitting our output to
	buffer    *bytes.Buffer // contains all bytes written up to the next new line
}

func newBufferedProcessOutput(w io.Writer) io.Writer {
	b := new(bytes.Buffer)
	return &bufferedWriter{
		Writer: w,
		buffer: b,
	}
}

func (o *bufferedWriter) Write(p []byte) (int, error) {
	for i, r := range p {
		err := o.buffer.WriteByte(r)
		if err != nil {
			return i, err
		}

		if r == '\n' {
			_, err := io.Copy(o.Writer, o.buffer)
			if err != nil {
				return i, err
			}
		}
	}

	return len(p), nil
}

// a processAutoDetectOutput attempts to detect the log format (e.g. JSON or
// plain) from the first message it receives and then either prints output
// unchanged or delegates it to its processJSONOutput.
type processAutoDetectOutput struct {
	io.Writer
	mu   sync.Mutex
	conf StructuredOutput
}

func newProcessAutoDetectOutput(w io.Writer, conf StructuredOutput) *processAutoDetectOutput {
	conf.Format = "auto"
	return &processAutoDetectOutput{
		Writer: w,
		conf:   conf,
	}
}

func (o *processAutoDetectOutput) Write(line []byte) (int, error) {
	o.mu.Lock()
	if o.conf.Format == "auto" {
		o.detectFormat(line)
	}
	o.mu.Unlock()

	return o.Writer.Write(line)
}

func (o *processAutoDetectOutput) detectFormat(line []byte) {
	m := map[string]interface{}{}
	err := json.Unmarshal(line, &m)
	switch {
	case err != nil:
		o.conf.Format = "unknown"
	case err == nil:
		o.conf.Format = "json"
		o.Writer = newProcessJSONOutput(o.Writer, o.conf)
	}
}

// a processJSONOutput is an io.Writer for processes which emit structured JSON
// messages. This writer expects that it will always receive complete json
// encoded messages on each write. Thus it is usually best to wrap each
// processJSONOutput into a bufferedWriter.
type processJSONOutput struct {
	io.Writer    // the writer we are eventually emitting our formatted output to
	messageField string
	levelField   string

	// taggingRules is a list of functions that tag a structured log message
	taggingRules []func(map[string]interface{}) (tag string)

	// tagActions maps tags to the action that should be applied to the tagged message
	tagActions map[string]tagAction
}

type tagAction struct {
	color color
}

func newProcessJSONOutput(w io.Writer, conf StructuredOutput) *processJSONOutput {
	o := &processJSONOutput{
		Writer:       w,
		messageField: conf.MessageField,
		levelField:   conf.LevelField,
	}

	for _, r := range conf.TaggingRules {
		o.addTaggingRule(r.Field, r.Value, r.Tag)
	}

	for tag, c := range conf.TagColors {
		o.setTagAction(tag, tagAction{color: parseColor(c)})
	}

	return o
}

var valueRegex = regexp.MustCompile("/(.+)/(.*)")

// addTaggingRule adds a new tagging rule to o. The `tag` is applied to each
// message which contains a certain `field` where the corresponding value is
// equal to the given `value`. Optionally the value can be a regular expression
// by surrounding it with slashes (e.g. /foo|bar/i).
func (o *processJSONOutput) addTaggingRule(field, value, tag string) {
	var re *regexp.Regexp
	if matches := valueRegex.FindStringSubmatch(value); matches != nil {
		reStr := matches[1]
		if strings.ContainsRune(matches[2], 'i') {
			reStr = "(?i)" + reStr
		}
		re, _ = regexp.Compile(reStr)
	}

	o.taggingRules = append(o.taggingRules, func(m map[string]interface{}) string {
		if re != nil && re.MatchString(o.stringField(m, field)) {
			return tag
		}

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

	_, err = o.Writer.Write([]byte(msg + "\n"))
	return len(line), err
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

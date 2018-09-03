package prox

import (
	"bytes"
	"fmt"
	"io"
	"strings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("output", func() {
	Describe("next", func() {
		It("should create a colorized process output with the correct prefix", func() {
			buffer := new(bytes.Buffer)
			o := &output{
				writer:       newSyncWriter(buffer),
				colors:       &colorPalette{colors: []color{colorYellow}},
				prefixLength: 8,
			}

			po := o.next(Process{Name: "test"})
			po.Write([]byte("This is a log message\n"))

			prefix := colorDefault + colorBold + colorYellow + "test     │ " + colorDefault
			Expect(buffer.String()).To(BeEquivalentTo(prefix + "This is a log message\n"))
		})

		Describe("without colors", func() {
			It("should not colorize the output", func() {
				buffer := new(bytes.Buffer)
				o := &output{
					writer:       newSyncWriter(buffer),
					colors:       &colorPalette{}, // color palette is empty
					prefixLength: 8,
				}

				po := o.next(Process{Name: "test"})
				po.Write([]byte("This is a log message\n"))

				Expect(buffer.String()).To(Equal("test     │ This is a log message\n"))
			})
		})
	})
})

var _ = Describe("multiWriter", func() {
	Describe("adding and removing writers", func() {
		It("should duplicate all messages to all registered writers", func() {
			w1, w2, w3 := new(bytes.Buffer), new(bytes.Buffer), new(bytes.Buffer)
			var o multiWriter

			o.AddWriter(w1)
			o.AddWriter(w2)
			o.AddWriter(w3)

			o.Write([]byte("Log message 1"))
			Expect(w1.String()).To(Equal("Log message 1"))
			Expect(w2.String()).To(Equal("Log message 1"))
			Expect(w2.String()).To(Equal("Log message 1"))

			w1.Reset()
			w2.Reset()
			w3.Reset()

			o.RemoveWriter(w2)
			o.Write([]byte("Log message 2"))
			Expect(w1.String()).To(Equal("Log message 2"))
			Expect(w2.String()).To(BeEmpty())
			Expect(w3.String()).To(Equal("Log message 2"))

			w1.Reset()
			w3.Reset()

			o.RemoveWriter(w3)
			o.Write([]byte("Log message 3"))
			Expect(w1.String()).To(Equal("Log message 3"))
			Expect(w2.String()).To(BeEmpty())
			Expect(w3.String()).To(BeEmpty())
		})

		PIt("should duplicate all messages to all writers even if some writers fail", func() {
			// It may happen that we fail to write to a connected client for some reason.
			// In this case we must ensure that we keep emitted log output in the main prox process.

			// TODO: implement test (code is already implemented)
		})
	})
})

var _ = Describe("processJSONOutput", func() {
	It("should output the message as well readable text", func() {
		w := new(bytes.Buffer)
		o := newProcessJSONOutput(w, StructuredOutput{MessageField: "message", LevelField: "level"})

		writeLine(o, `{"level": "info", "message": "Hello World", "foo": "bar"}`)
		writeLine(o, `{"level": "info", "message": "An error has occurred", "n":42, "object": {"test":true}}`)

		Expect(w.String()).To(Equal(strings.Join([]string{
			"[INFO]\tHello World\t" + `{ "foo": "bar" }`,
			"[INFO]\tAn error has occurred\t" + `{ "n": 42, "object": { "test": true } }`,
		}, "\n") + "\n"))
	})

	It("should color messages based on the parsed log level", func() {
		w := new(bytes.Buffer)
		o := newProcessJSONOutput(w, StructuredOutput{MessageField: "message", LevelField: "level"})
		o.addTaggingRule("level", "error", "error")
		o.setTagAction("error", tagAction{color: colorRed})

		writeLine(o, `{"level": "info",  "message": "Hello World"}`)
		writeLine(o, `{"level": "error", "message": "An error has occurred"}`)

		Expect(w.String()).To(Equal(strings.Join([]string{
			"[INFO]\tHello World",
			colored(colorRed, "[ERROR]\tAn error has occurred"),
		}, "\n") + "\n"))
	})

	It("should color messages using regular expressions", func() {
		w := new(bytes.Buffer)
		o := newProcessJSONOutput(w, StructuredOutput{MessageField: "message", LevelField: "level"})
		o.addTaggingRule("message", "/t..t/", "my-tag")
		o.setTagAction("my-tag", tagAction{color: colorBlue})

		writeLine(o, `{"level": "info",  "message": "The test is a lie"}`)
		writeLine(o, `{"level": "error", "message": "An error has occurred"}`)

		Expect(w.String()).To(Equal(strings.Join([]string{
			colored(colorBlue, "[INFO]\tThe test is a lie"),
			"[ERROR]\tAn error has occurred",
		}, "\n") + "\n"))
	})

	It("should color messages using regular expressions supporting case insensitive matching", func() {
		w := new(bytes.Buffer)
		o := newProcessJSONOutput(w, StructuredOutput{
			MessageField: "message",
			LevelField:   "level",
			TaggingRules: []TaggingRule{
				{Field: "message", Value: "/t..t/i", Tag: "my-tag"},
			},
			TagColors: map[string]string{
				"my-tag": "blue",
			},
		})

		writeLine(o, `{"level": "info",  "message": "This is a tEsT"}`)
		writeLine(o, `{"level": "error", "message": "An error has occurred"}`)

		Expect(w.String()).To(Equal(strings.Join([]string{
			colored(colorBlue, "[INFO]\tThis is a tEsT"),
			"[ERROR]\tAn error has occurred",
		}, "\n") + "\n"))
	})

	Describe("weird input", func() {
		It("should not crash if the message field is not present", func() {
			w := new(bytes.Buffer)
			o := newProcessJSONOutput(w, StructuredOutput{MessageField: "message", LevelField: "level"})

			writeLine(o, `{"level": "info"}`)
			writeLine(o, `{"level": "info", "n":42, "object": {"test":true}}`)

			Expect(w.String()).To(Equal(strings.Join([]string{
				"[INFO]\t",
				"[INFO]\t\t" + `{ "n": 42, "object": { "test": true } }`,
			}, "\n") + "\n"))
		})

		It("should not crash if the level field is not present", func() {
			w := new(bytes.Buffer)
			o := newProcessJSONOutput(w, StructuredOutput{MessageField: "message", LevelField: "level"})

			writeLine(o, `{"message": "Hello World", "foo": "bar"}`)
			writeLine(o, `{"message": "An error has occurred", "n":42, "object": {"test":true}}`)

			Expect(w.String()).To(Equal(strings.Join([]string{
				"Hello World\t" + `{ "foo": "bar" }`,
				"An error has occurred\t" + `{ "n": 42, "object": { "test": true } }`,
			}, "\n") + "\n"))
		})
	})
})

var _ = Describe("processAutoDetectOutput", func() {
	It("should automatically detect JSON output", func() {
		w := new(bytes.Buffer)
		o := newProcessAutoDetectOutput(w, StructuredOutput{MessageField: "message", LevelField: "level"})

		writeLine(o, `{"level": "info", "message": "Hello World", "foo": "bar"}`)
		writeLine(o, `{"level": "info", "message": "An error has occurred", "n":42, "object": {"test":true}}`)

		Expect(w.String()).To(Equal(strings.Join([]string{
			"[INFO]\tHello World\t" + `{ "foo": "bar" }`,
			"[INFO]\tAn error has occurred\t" + `{ "n": 42, "object": { "test": true } }`,
		}, "\n") + "\n"))
	})

	It("should print non-JSON output normally", func() {
		w := new(bytes.Buffer)
		o := newProcessAutoDetectOutput(w, StructuredOutput{MessageField: "message", LevelField: "level"})

		writeLine(o, "This is an unstructured message. It should be printed unchanged")
		writeLine(o, `{"level": "info", "message": "If there is JSON output later we still print it normally"}`)
		writeLine(o, "Another message")

		Expect(w.String()).To(Equal(strings.Join([]string{
			"This is an unstructured message. It should be printed unchanged",
			`{"level": "info", "message": "If there is JSON output later we still print it normally"}`,
			"Another message",
		}, "\n") + "\n"))
	})
})

var _ = Describe("bufferedWriter", func() {
	It("should print complete lines", func() {
		out := new(bytes.Buffer)
		w := newBufferedProcessOutput(out)
		fmt.Fprint(w, "This is a complete line\n")
		Expect(out.String()).To(Equal("This is a complete line\n"))
	})

	It("should buffer output until the newline is complete", func() {
		out := new(bytes.Buffer)
		w := newBufferedProcessOutput(out)
		fmt.Fprint(w, "This")
		Expect(out.String()).To(BeEmpty())
		fmt.Fprint(w, " is a ")
		Expect(out.String()).To(BeEmpty())
		fmt.Fprint(w, "comp")
		Expect(out.String()).To(BeEmpty())
		fmt.Fprint(w, "lete line\n")
		Expect(out.String()).To(Equal("This is a complete line\n"))
	})

	It("should work with multiple lines", func() {
		out := new(bytes.Buffer)
		w := newBufferedProcessOutput(out)
		fmt.Fprint(w, "This is line one\nAnd this is line two\n33333\n")
		Expect(out.String()).To(Equal("This is line one\nAnd this is line two\n33333\n"))
	})

	It("should work with windows line endings (CRLF)", func() {
		out := new(bytes.Buffer)
		w := newBufferedProcessOutput(out)
		fmt.Fprint(w, "This is line")
		Expect(out.String()).To(BeEmpty())
		fmt.Fprint(w, " one\r\n")
		Expect(out.String()).To(Equal("This is line one\r\n"))

		out.Reset()
		fmt.Fprint(w, "And line two")
		Expect(out.String()).To(BeEmpty())
		fmt.Fprint(w, "\r\n")
		Expect(out.String()).To(Equal("And line two\r\n"))
	})
})

func writeLine(w io.Writer, s string) {
	_, err := w.Write([]byte(s + "\n"))
	Expect(err).NotTo(HaveOccurred())
}

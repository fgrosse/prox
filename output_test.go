package prox

import (
	"bytes"
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
				writer:       buffer,
				colors:       &colorPalette{colors: []color{colorYellow}},
				prefixLength: 8,
			}

			po := o.next("test")
			po.Write([]byte("This is a log message"))

			prefix := colorDefault + colorBold + colorYellow + "test     │ " + colorDefault
			Expect(buffer.String()).To(BeEquivalentTo(prefix + "This is a log message\n"))
		})

		Describe("without colors", func() {
			It("should not colorize the output", func() {
				buffer := new(bytes.Buffer)
				o := &output{
					writer:       buffer,
					colors:       &colorPalette{}, // color palette is empty
					prefixLength: 8,
				}

				po := o.next("test")
				po.Write([]byte("This is a log message"))

				Expect(buffer.String()).To(Equal("test     │ This is a log message\n"))
			})
		})
	})
})

var _ = Describe("processOutput", func() {
	Describe("adding and removing writers", func() {
		It("should duplicate all messages to all registered writers", func() {
			w1, w2, w3 := new(bytes.Buffer), new(bytes.Buffer), new(bytes.Buffer)
			o := newProcessOutput(w1)

			o.AddWriter(w2)
			o.AddWriter(w3)

			o.Write([]byte("Log message 1"))
			Expect(w1.String()).To(Equal("Log message 1\n"))
			Expect(w2.String()).To(Equal("Log message 1\n"))
			Expect(w2.String()).To(Equal("Log message 1\n"))

			w1.Reset()
			w2.Reset()
			w3.Reset()

			o.RemoveWriter(w2)
			o.Write([]byte("Log message 2"))
			Expect(w1.String()).To(Equal("Log message 2\n"))
			Expect(w2.String()).To(BeEmpty())
			Expect(w3.String()).To(Equal("Log message 2\n"))

			w1.Reset()
			w3.Reset()

			o.RemoveWriter(w3)
			o.Write([]byte("Log message 3"))
			Expect(w1.String()).To(Equal("Log message 3\n"))
			Expect(w2.String()).To(BeEmpty())
			Expect(w3.String()).To(BeEmpty())
		})

		PIt("should duplicate all messages to all writers even if some writers fail", func() {
			// It may happen that we fail to write to a connected client for some reason.
			// In this case we must ensure that we keep emitted log output in the main prox process.
			// The issue is that the currently used multiwriter will fail upon the first error.
		})
	})
})

var _ = Describe("processJSONOutput", func() {
	writeLine := func(w io.Writer, s string) {
		_, err := w.Write([]byte(s + "\n"))
		Expect(err).NotTo(HaveOccurred())
	}

	It("should output the message as well readable text", func() {
		w := new(bytes.Buffer)
		o := newProcessJSONOutput(w)

		writeLine(o, `{"level": "info", "message": "Hello World", "foo": "bar"}`)
		writeLine(o, `{"level": "info", "message": "An error has occurred", "n":42, "object": {"test":true}}`)

		Expect(w.String()).To(Equal(strings.Join([]string{
			"[INFO]\tHello World\t" + `{ "foo": "bar" }`,
			"[INFO]\tAn error has occurred\t" + `{ "n": 42, "object": { "test": true } }`,
		}, "\n") + "\n"))
	})

	It("should color messages based on the parsed log level", func() {
		w := new(bytes.Buffer)
		o := newProcessJSONOutput(w)

		writeLine(o, `{"level": "info", "message": "Hello World"}`)
		writeLine(o, `{"level": "error", "message": "An error has occurred"}`)

		Expect(w.String()).To(Equal(strings.Join([]string{
			"[INFO]\tHello World",
			colored(colorRed, "[ERROR]\tAn error has occurred"),
		}, "\n") + "\n"))
	})

	Describe("weird input", func() {
		It("should not crash if the message field is not present", func() {
			w := new(bytes.Buffer)
			o := newProcessJSONOutput(w)

			writeLine(o, `{"level": "info"}`)
			writeLine(o, `{"level": "info", "n":42, "object": {"test":true}}`)

			Expect(w.String()).To(Equal(strings.Join([]string{
				"[INFO]",
				"[INFO]\t" + `{ "n": 42, "object": { "test": true } }`,
			}, "\n") + "\n"))
		})

		It("should not crash if the level field is not present", func() {
			w := new(bytes.Buffer)
			o := newProcessJSONOutput(w)

			writeLine(o, `{"message": "Hello World", "foo": "bar"}`)
			writeLine(o, `{"message": "An error has occurred", "n":42, "object": {"test":true}}`)

			Expect(w.String()).To(Equal(strings.Join([]string{
				"Hello World\t" + `{ "foo": "bar" }`,
				"An error has occurred\t" + `{ "n": 42, "object": { "test": true } }`,
			}, "\n") + "\n"))
		})
	})
})

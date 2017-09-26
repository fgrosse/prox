package prox

import (
	"bytes"

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

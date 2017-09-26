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

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
				writer: buffer,
				colors: &colorPalette{colors: []color{colorYellow}},
			}

			po := o.next("test", 8)
			po.Write([]byte("This is a log message"))

			prefix := colorDefault + colorBold + colorYellow + "test     │ " + colorDefault
			Expect(buffer.String()).To(BeEquivalentTo(prefix + "This is a log message\n"))
		})

		PDescribe("without colors", func() {
			It("should not colorize the output", func() {
				buffer := new(bytes.Buffer)
				o := &output{
					writer: buffer,
					colors: &colorPalette{}, // color palette is empty
				}

				po := o.next("test", 8)
				po.Write([]byte("This is a log message"))

				Expect(buffer.String()).To(Equal("test     │ This is a log message\n"))
			})
		})
	})
})

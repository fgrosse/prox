package prox

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("output", func() {
	Describe("next", func() {
		It("should create a colorized prefix of the correct length", func() {
			o := &output{
				writer: GinkgoWriter,
				colors: &colorPalette{colors: []color{colorYellow}},
			}

			po := o.next("test", 8)
			Expect(po.prefix).To(BeEquivalentTo(colorDefault + colorBold + colorYellow + "test     │" + colorDefault))
		})
	})
})

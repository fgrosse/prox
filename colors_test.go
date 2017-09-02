package prox

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("colorProvider", func() {
	Describe("next", func() {
		It("should return a random permutation of colors", func() {
			cp := newColorProvider()
			Expect(len(cp.colors)).To(BeNumerically(">", 2))

			// fetch all available colors
			colors := map[color]struct{}{}
			for i := 0; i < len(cp.colors); i++ {
				c := cp.next()
				Expect(colors).NotTo(HaveKey(c), "there should be no duplicate colors")
				colors[c] = struct{}{}
			}
		})

		It("should reuse its colors when asked for more colors than we have defined", func() {
			cp := newColorProvider()
			colors1 := map[color]struct{}{}
			colors2 := map[color]struct{}{}

			for i := 0; i < len(cp.colors); i++ {
				c := cp.next()
				colors1[c] = struct{}{}
			}

			for i := 0; i < len(cp.colors); i++ {
				c := cp.next()
				colors2[c] = struct{}{}
			}

			Expect(colors1).To(Equal(colors2))
		})
	})
})

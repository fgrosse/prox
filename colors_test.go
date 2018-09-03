package prox

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("colorPalette", func() {
	Describe("next", func() {
		It("should return a random permutation of colors", func() {
			cp := newColorPalette()
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
			cp := newColorPalette()
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

var _ = Describe("parseColor", func() {
	It("should support bold for all colors", func() {
		colors := map[string]color{
			"red": colorRed,
			"green": colorGreen,
			"yellow": colorYellow,
			"blue": colorBlue,
			"magenta": colorMagenta,
			"cyan": colorCyan,
			"white": colorWhite,
		}

		for colorStr, expected := range colors {
			actual := parseColor(colorStr)
			Expect(actual).To(Equal(expected), "Parsing " + colorStr)

			actual = parseColor(colorStr+"-bold")
			Expect(actual).To(Equal(expected+colorBold), "Parsing " + colorStr+ " (bold)")
		}
	})
})

package prox

import "fmt"

type color string

const (
	colorNone    color = ""
	colorDefault color = "\x1b[0m"
	colorBold    color = "\x1b[1m"
	colorRed     color = "\x1b[31m"
	colorGreen   color = "\x1b[32m"
	colorYellow  color = "\x1b[33m"
	colorBlue    color = "\x1b[34m"
	colorMagenta color = "\x1b[35m"
	colorCyan    color = "\x1b[36m"
	colorWhite   color = "\x1b[37m" // reserved for the prox output
)

type colorPalette struct {
	colors []color
	i      int
}

func newColorPalette() *colorPalette {
	return &colorPalette{colors: []color{
		colorCyan,
		colorYellow,
		colorGreen,
		colorMagenta,
		colorRed,
		colorBlue,
	}}
}

func (p *colorPalette) next() color {
	if p == nil || len(p.colors) == 0 {
		return colorNone
	}

	c := p.colors[p.i]
	p.i++
	if p.i >= len(p.colors) {
		p.i = 0
	}

	return c
}

func colored(c color, s string) string {
	return fmt.Sprint(c, s, colorDefault)
}

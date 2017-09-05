package prox

import (
	"sync"
)

type color string

const (
	colorDefault color = "\x1b[0m"
	colorBold    color = "\x1b[1m"
	colorRed     color = "\x1b[31m"
	colorGreen   color = "\x1b[32m"
	colorYellow  color = "\x1b[33m"
	colorBlue    color = "\x1b[34m"
	colorMagenta color = "\x1b[35m"
	colorCyan    color = "\x1b[36m"
	colorWhite   color = "\x1b[37m"
)

// colors are all colors that are used to distinguish the output of all
// processes. This list is ordered such that the first used color is first.
var colors = []color{
	// colorWhite, TODO: use for prox output
	colorCyan,
	colorYellow,
	colorGreen,
	colorMagenta,
	colorRed,
	colorBlue,
}

type colorProvider struct {
	mu     sync.Mutex
	colors []color
	i      int
}

func newColorProvider() *colorProvider {
	return &colorProvider{colors: colors}
}

func (p *colorProvider) next() color {
	p.mu.Lock()
	c := p.colors[p.i]
	p.i++
	if p.i >= len(p.colors) {
		p.i = 0
	}
	p.mu.Unlock()

	return c
}

package prox

import (
	"fmt"
	"math/rand"
	"sync"
)

type color string

const (
	colorDefault   color = "\x1b[0m"
	colorGreen     color = "\x1b[32m"
	colorYellow    color = "\x1b[33m"
	colorLightBlue color = "\x1b[34m"
	colorPurple    color = "\x1b[35m"
	colorCyan      color = "\x1b[36m"
	colorLightGray color = "\x1b[37m"
	colorGray      color = "\x1b[90m"
	colorRed       color = "\x1b[91m"
)

func (c color) apply(s string) string {
	return fmt.Sprint(c, s, colorDefault)
}

type colorProvider struct {
	mu     sync.Mutex
	colors []color
}

func newColorProvider() *colorProvider {
	all := []color{
		colorGreen, colorYellow, colorLightBlue, colorPurple,
		colorCyan, colorLightGray, colorGray, colorRed,
	}

	colors := make([]color, len(all))
	for i, j := range rand.Perm(len(all)) {
		colors[i] = all[j]
	}

	return &colorProvider{colors: colors}
}

func (p *colorProvider) next() color {
	p.mu.Lock()
	c := p.colors[0]
	p.colors = append(p.colors[1:], c)
	p.mu.Unlock()

	return c
}

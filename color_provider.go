package prox

import (
	"math/rand"
	"sync"
)

type color string

const (
	DefaultStyle   color = "\x1b[0m"
	GreenColor     color = "\x1b[32m"
	YellowColor    color = "\x1b[33m"
	LightBlueColor color = "\x1b[34m"
	PurpleColor    color = "\x1b[35m"
	CyanColor      color = "\x1b[36m"
	LightGrayColor color = "\x1b[37m"
	GrayColor      color = "\x1b[90m"
	RedColor       color = "\x1b[91m"
)

type colorProvider struct {
	mu     sync.Mutex
	colors []color
}

func newColorProvider() *colorProvider {
	all := []color{
		GreenColor,
		YellowColor,
		LightBlueColor,
		PurpleColor,
		CyanColor,
		LightGrayColor,
		GrayColor,
		RedColor,
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

package prox

type Process interface {
	Name() string
	Run() error // TODO pass a ctx
	Interrupt() error
}

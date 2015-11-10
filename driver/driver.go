package driver

type Driver interface {
	Probe() error
	Close()
}

type Noop struct{}

func (n *Noop) Probe() error {
	return nil
}

func (n *Noop) Close() {
	// NOOP
}

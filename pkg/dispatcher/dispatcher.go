package dispatcher

type Dispatcher struct{}

func New() *Dispatcher {
	return &Dispatcher{}
}

func (d *Dispatcher) Dispatch(_ []any) {}

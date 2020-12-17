package proc

type ProcFunc func(<-chan struct{}, chan<- error)

type Proc struct {
	funcs  []ProcFunc
	stopCs []chan struct{}
	errC   chan error
}

func NewProc() *Proc {
	proc := &Proc{
		funcs: make([]ProcFunc, 0),
		errC:  make(chan error),
	}
	return proc
}

func (p *Proc) Start() <-chan error {
	p.stopCs = make([]chan struct{}, 0, len(p.funcs))
	for _, _func := range p.funcs {
		stopC := make(chan struct{})
		go _func(stopC, p.errC)
		p.stopCs = append(p.stopCs, stopC)
	}
	return p.errC
}

func (p *Proc) Stop() {
	for _, stop := range p.stopCs {
		stop <- struct{}{}
	}
	return
}

func (p *Proc) Add(_func ProcFunc) { p.funcs = append(p.funcs, _func) }

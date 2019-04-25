package pmgo

import mgo "gopkg.in/mgo.v2"

type PipeManager interface {
	All(result interface{}) error
	AllowDiskUse() PipeManager
	Batch(n int) PipeManager
	Explain(result interface{}) error
	Iter() *mgo.Iter
	One(result interface{}) error
}

type Pipe struct {
	pipe *mgo.Pipe
}

func NewPipeManager(p *mgo.Pipe) PipeManager {
	return &Pipe{
		pipe: p,
	}
}

func (p *Pipe) All(result interface{}) error {
	return p.pipe.All(result)
}

func (p *Pipe) AllowDiskUse() PipeManager {
	return &Pipe{
		pipe: p.pipe,
	}
}

func (p *Pipe) Batch(n int) PipeManager {
	return &Pipe{
		pipe: p.pipe.Batch(n),
	}
}

func (p *Pipe) Explain(result interface{}) error {
	return p.pipe.Explain(result)
}

func (p *Pipe) Iter() *mgo.Iter {
	return p.pipe.Iter()
}

func (p *Pipe) One(result interface{}) error {
	return p.pipe.One(result)
}

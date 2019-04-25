package pmgo

import mgo "gopkg.in/mgo.v2"

type IterManager interface {
	All(result interface{}) error
	Close() error
	Done() bool
	Err() error
	For(result interface{}, f func() error) (err error)
	Next(result interface{}) bool
	Timeout() bool
}

type Iter struct {
	iter *mgo.Iter
}

func NewIter(iter *mgo.Iter) IterManager {
	return &Iter{iter}
}

func (i *Iter) All(result interface{}) error {
	return i.iter.All(result)
}

func (i *Iter) Close() error {
	return i.iter.Close()
}

func (i *Iter) Done() bool {
	return i.iter.Done()
}

func (i *Iter) Err() error {
	return i.iter.Err()
}

func (i *Iter) For(result interface{}, f func() error) (err error) {
	return i.iter.For(result, f)
}

func (i *Iter) Next(result interface{}) bool {
	return i.iter.Next(result)
}

func (i *Iter) Timeout() bool {
	return i.iter.Timeout()
}

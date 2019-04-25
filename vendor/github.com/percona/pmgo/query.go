package pmgo

import (
	"time"

	mgo "gopkg.in/mgo.v2"
)

type QueryManager interface {
	All(result interface{}) error
	Apply(change mgo.Change, result interface{}) (info *mgo.ChangeInfo, err error)
	Batch(n int) QueryManager
	Comment(comment string) QueryManager
	Count() (n int, err error)
	Distinct(key string, result interface{}) error
	Explain(result interface{}) error
	For(result interface{}, f func() error) error
	Hint(indexKey ...string) QueryManager
	Iter() *mgo.Iter
	Limit(n int) QueryManager
	LogReplay() QueryManager
	MapReduce(job *mgo.MapReduce, result interface{}) (info *mgo.MapReduceInfo, err error)
	One(result interface{}) (err error)
	Prefetch(p float64) QueryManager
	Select(selector interface{}) QueryManager
	SetMaxScan(n int) QueryManager
	SetMaxTime(d time.Duration) QueryManager
	Skip(n int) QueryManager
	Snapshot() QueryManager
	Sort(fields ...string) QueryManager
	Tail(timeout time.Duration) IterManager
}
type Query struct {
	query *mgo.Query
}

func NewQueryManager(q *mgo.Query) QueryManager {
	return &Query{
		query: q,
	}
}

func (q *Query) All(result interface{}) error {
	return q.query.All(result)
}

func (q *Query) Apply(change mgo.Change, result interface{}) (info *mgo.ChangeInfo, err error) {
	return q.query.Apply(change, result)
}

func (q *Query) Batch(n int) QueryManager {
	return &Query{q.query.Batch(n)}
}

func (q *Query) Comment(comment string) QueryManager {
	return &Query{q.query.Comment(comment)}
}

func (q *Query) Count() (int, error) {
	return q.query.Count()
}

func (q *Query) Distinct(key string, result interface{}) error {
	return q.query.Distinct(key, result)
}

func (q *Query) Explain(result interface{}) error {
	return q.query.Explain(result)
}

func (q *Query) For(result interface{}, f func() error) error {
	return q.query.For(result, f)
}

func (q *Query) Hint(indexKey ...string) QueryManager {
	return &Query{q.query.Hint(indexKey...)}
}

func (q *Query) Iter() *mgo.Iter {
	return q.query.Iter()
}

func (q *Query) Limit(n int) QueryManager {
	return &Query{q.query.Limit(n)}
}

func (q *Query) LogReplay() QueryManager {
	return &Query{q.query.LogReplay()}
}

func (q *Query) MapReduce(job *mgo.MapReduce, result interface{}) (info *mgo.MapReduceInfo, err error) {
	return q.query.MapReduce(job, result)
}

func (q *Query) One(result interface{}) error {
	return q.query.One(result)
}

func (q *Query) Prefetch(p float64) QueryManager {
	return &Query{q.query.Prefetch(p)}
}
func (q *Query) Select(selector interface{}) QueryManager {
	return &Query{q.query.Select(selector)}
}

func (q *Query) SetMaxScan(n int) QueryManager {
	return &Query{q.query.SetMaxScan(n)}
}

func (q *Query) SetMaxTime(d time.Duration) QueryManager {
	return &Query{q.query.SetMaxTime(d)}
}

func (q *Query) Skip(n int) QueryManager {
	return &Query{q.query.Skip(n)}
}

func (q *Query) Snapshot() QueryManager {
	return &Query{q.query.Snapshot()}
}

func (q *Query) Sort(fields ...string) QueryManager {
	return &Query{q.query.Sort(fields...)}
}

func (q *Query) Tail(timeout time.Duration) IterManager {
	return q.query.Tail(timeout)
}

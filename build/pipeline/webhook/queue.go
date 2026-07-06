package main

import (
	"errors"
	"sync"
)

// Queue serializes build jobs so only one runs at a time. The buffer size
// caps queued-but-not-yet-started jobs; deeper queues tend to be stale when
// they finally run.
type Queue struct {
	ch     chan Job
	runner *Runner
	wg     sync.WaitGroup
}

func NewQueue(r *Runner, size int) *Queue {
	return &Queue{
		ch:     make(chan Job, size),
		runner: r,
	}
}

func (q *Queue) Enqueue(job Job) error {
	select {
	case q.ch <- job:
		return nil
	default:
		return errors.New("queue is full")
	}
}

func (q *Queue) Run() {
	q.wg.Add(1)
	defer q.wg.Done()
	for job := range q.ch {
		q.runner.Run(job)
	}
}

func (q *Queue) Close() {
	close(q.ch)
	q.wg.Wait()
}

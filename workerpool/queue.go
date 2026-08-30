package workerpool

import (
	"slices"
	"sync"
)

// MemoryQueue is an in-memory FIFO that implements Queue.
type MemoryQueue struct {
	mu        sync.Mutex
	ready     []Job
	completed []string
}

func NewMemoryQueue(jobs ...Job) *MemoryQueue {
	return &MemoryQueue{ready: slices.Clone(jobs)}
}

func (q *MemoryQueue) Reserve() (*Job, bool) {
	q.mu.Lock()
	defer q.mu.Unlock()
	if len(q.ready) == 0 {
		return nil, false
	}
	job := q.ready[0]
	q.ready = q.ready[1:]
	return &job, true
}

func (q *MemoryQueue) Complete(id string) {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.completed = append(q.completed, id)
}

func (q *MemoryQueue) Fail(id string) {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.ready = append(q.ready, Job{ID: id})
}

func (q *MemoryQueue) Completed() []string {
	q.mu.Lock()
	defer q.mu.Unlock()
	return slices.Clone(q.completed)
}

package main

import "fmt"

type state string

const (
	StatePending state = "pending"
	StateRunning state = "running"
	StateDone    state = "done"
	StateFailed  state = "failed"
)

type Job struct {
	ID       string
	Payload  string
	Attempts int
	State    state
}

type Queue struct {
	maxRetries int
	jobs       map[string]Job
	ready      []string
	dlq        []string
}

func New(maxRetries int) *Queue {
	return &Queue{
		maxRetries: maxRetries,
		jobs:       make(map[string]Job),
		ready:      make([]string, 0),
		dlq:        make([]string, 0),
	}
}

func (q *Queue) Enqueue(id string, payload string) {
	if _, exists := q.jobs[id]; exists {
		return
	}

	job := Job{
		ID:       id,
		Payload:  payload,
		Attempts: 0,
		State:    StatePending,
	}
	q.jobs[id] = job
	q.ready = append(q.ready, id)
}

func (q *Queue) Reserve() (*Job, bool) {
	if len(q.ready) == 0 {
		return nil, false
	}

	id := q.ready[0]
	q.ready = q.ready[1:]
	job := q.jobs[id]
	job.State = StateRunning
	q.jobs[id] = job
	return &job, true
}

func (q *Queue) Complete(id string) {
	if id == "" || q.jobs[id].State != StateRunning || q.jobs[id].ID != id {
		fmt.Printf("invalid job id: %s", id)
		return
	}

	job := q.jobs[id]
	job.State = StateDone
	q.jobs[id] = job
}

func (q *Queue) Fail(id string) {
	if id == "" || q.jobs[id].State != StateRunning || q.jobs[id].ID != id {
		fmt.Printf("invalid job id: %s", id)
		return
	}

	job := q.jobs[id]
	job.Attempts++
	if job.Attempts >= q.maxRetries {
		job.State = StateFailed
		q.jobs[id] = job
		q.dlq = append(q.dlq, id)
		return
	}
	job.State = StatePending
	q.jobs[id] = job
	q.ready = append(q.ready, id)
}

func (q *Queue) DLQ() []string {
	return q.dlq
}

func main() {
	q := New(2)
	q.Enqueue("job-a", "send email")
	q.Enqueue("job-b", "resize image")
	fmt.Println("enqueued job-a, job-b  maxRetries=2")

	job, ok := q.Reserve()
	mustReserve(ok, job)
	fmt.Printf("reserved %s  state=%s  attempts=%d\n", job.ID, q.jobs[job.ID].State, q.jobs[job.ID].Attempts)
	q.Fail(job.ID)
	fmt.Printf("failed %s → retry  ready=%v  dlq=%v\n", job.ID, q.ready, q.DLQ())
	printJob(q, "job-a")

	job, ok = q.Reserve()
	mustReserve(ok, job)
	fmt.Printf("reserved %s  state=%s\n", job.ID, q.jobs[job.ID].State)
	q.Complete(job.ID)
	fmt.Printf("completed %s  state=%s\n", job.ID, q.jobs[job.ID].State)

	job, ok = q.Reserve()
	mustReserve(ok, job)
	fmt.Printf("reserved %s  attempts=%d\n", job.ID, q.jobs[job.ID].Attempts)
	q.Fail(job.ID)
	fmt.Printf("failed %s → dlq  ready=%v  dlq=%v\n", job.ID, q.ready, q.DLQ())
	printJob(q, "job-a")

	_, ok = q.Reserve()
	fmt.Printf("reserve on empty queue: ok=%v\n", ok)
}

func mustReserve(ok bool, job *Job) {
	if !ok || job == nil {
		panic("expected a reserved job")
	}
}

func printJob(q *Queue, id string) {
	j := q.jobs[id]
	fmt.Printf("  %s  payload=%q  attempts=%d  state=%s\n", j.ID, j.Payload, j.Attempts, j.State)
}

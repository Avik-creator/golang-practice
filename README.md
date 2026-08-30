# golang-practice

Senior-style Go interview exercises. Implement them yourself; use the notes as a tracker.

## Done

- [x] **Finite concurrent web crawler** (`finiteconcurrentwebcrawler/`)
  - Start URL, visited set *before* `go`, semaphore around `Fetch`, `WaitGroup` so it terminates
  - MongoDB-style concurrency question
- [x] **Work queue + retry + DLQ** (`work_queue/`)
  - Store by ID, FIFO `ready`, `Reserve` / `Complete` / `Fail`
  - Retry until `maxRetries`, then dead-letter queue
  - OpenAI-style job-lifecycle question
- [x] **Rate limiter (token bucket)** (`ratelimiter/`)
  - Lazy refill, burst cap, `Allow` / `Wait(ctx)`
- [x] **Worker pool** (`workerpool/`)
  - N workers, `Queue` interface, cancel via `context`

## Up next

Pick one; same pattern as before (types → one method → tests).

- [ ] **LRU cache with TTL** — `Get` / `Set`, max size, expiry, mutex, `go test -race`

## Backlog

- [ ] **`singleflight`** — coalesced in-flight fetches; same key = one call, many waiters
- [ ] **Delayed job queue** — `runAt` min-heap + existing FIFO ready queue
- [ ] **In-memory KV with transactions** — `Get` / `Set` / `Begin` / `Commit` / `Rollback`
- [ ] **Pub/sub** — `Subscribe` / `Publish`; bounded buffers so slow subs don’t block forever
- [ ] **Circuit breaker** — closed → open → half-open; fail-fast while open

## Later (design-first, code a slice if asked)

- Consistent hashing
- Raft / replication
- Distributed lock
- Full HTTP crawler (HTML, same-host, robots)

## Run

```bash
# crawler
cd finiteconcurrentwebcrawler && go test -race -count=1 ./... && go run ./cmd/crawler

# work queue
cd work_queue && go run .

# rate limiter
cd ratelimiter && go test -race -count=1 . && go run ./cmd/ratelimiter

# worker pool
cd workerpool && go test -race -count=1 .
```

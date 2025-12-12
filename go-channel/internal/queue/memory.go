package queue

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/google/uuid"

	"go-channel/internal/types"
)

// MemoryQueue is a bounded, in-memory implementation of the Queue interface.
type MemoryQueue struct {
	jobCh     chan string
	mu        sync.RWMutex
	store     map[string]types.Job
	closed    bool
	closeOnce sync.Once
}

func NewMemoryQueue(size int) *MemoryQueue {
	if size <= 0 {
		size = 1
	}
	return &MemoryQueue{
		jobCh: make(chan string, size),
		store: make(map[string]types.Job),
	}
}

func (q *MemoryQueue) Enqueue(ctx context.Context, req types.EnqueueRequest) (types.Job, error) {
	q.mu.Lock()
	if q.closed {
		q.mu.Unlock()
		return types.Job{}, ErrQueueClosed
	}
	job := types.Job{
		ID:          uuid.NewString(),
		Payload:     req.Payload,
		Status:      types.JobStatusQueued,
		MaxAttempts: max(req.MaxAttempts, 3),
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}
	q.store[job.ID] = job
	q.mu.Unlock()

	select {
	case <-ctx.Done():
		return types.Job{}, ctx.Err()
	case q.jobCh <- job.ID:
		return job, nil
	}
}

func (q *MemoryQueue) Dequeue(ctx context.Context) (types.Job, error) {
	select {
	case <-ctx.Done():
		return types.Job{}, ctx.Err()
	case id, ok := <-q.jobCh:
		if !ok {
			return types.Job{}, ErrQueueClosed
		}
		q.mu.Lock()
		job, found := q.store[id]
		if !found {
			q.mu.Unlock()
			return types.Job{}, errors.New("job not found after dequeue")
		}
		job.Attempts++
		job.Status = types.JobStatusRunning
		job.UpdatedAt = time.Now().UTC()
		q.store[id] = job
		q.mu.Unlock()
		return job, nil
	}
}

func (q *MemoryQueue) Ack(ctx context.Context, jobID string) error {
	q.mu.Lock()
	defer q.mu.Unlock()
	job, ok := q.store[jobID]
	if !ok {
		return errors.New("job missing")
	}
	job.Status = types.JobStatusSucceeded
	job.ErrMsg = ""
	job.UpdatedAt = time.Now().UTC()
	q.store[jobID] = job
	return ctx.Err()
}

func (q *MemoryQueue) Fail(ctx context.Context, jobID string, cause error) (types.Job, error) {
	q.mu.Lock()
	defer q.mu.Unlock()
	job, ok := q.store[jobID]
	if !ok {
		return types.Job{}, errors.New("job missing")
	}
	job.Status = types.JobStatusFailed
	job.ErrMsg = cause.Error()
	job.UpdatedAt = time.Now().UTC()
	q.store[jobID] = job
	return job, ctx.Err()
}

func (q *MemoryQueue) Requeue(ctx context.Context, job types.Job) error {
	q.mu.Lock()
	if q.closed {
		q.mu.Unlock()
		return ErrQueueClosed
	}
	job.Status = types.JobStatusQueued
	job.UpdatedAt = time.Now().UTC()
	q.store[job.ID] = job
	q.mu.Unlock()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case q.jobCh <- job.ID:
		return nil
	}
}

func (q *MemoryQueue) Get(_ context.Context, id string) (types.Job, bool) {
	q.mu.RLock()
	defer q.mu.RUnlock()
	job, ok := q.store[id]
	return job, ok
}

func (q *MemoryQueue) Close() {
	q.closeOnce.Do(func() {
		q.mu.Lock()
		q.closed = true
		q.mu.Unlock()
		close(q.jobCh)
	})
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

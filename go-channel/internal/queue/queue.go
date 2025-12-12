package queue

import (
	"context"
	"errors"

	"go-channel/internal/types"
)

var ErrQueueClosed = errors.New("queue closed")

type Queue interface {
	Enqueue(ctx context.Context, req types.EnqueueRequest) (types.Job, error)
	Dequeue(ctx context.Context) (types.Job, error)
	Ack(ctx context.Context, jobID string) error
	Fail(ctx context.Context, jobID string, cause error) (types.Job, error)
	Requeue(ctx context.Context, job types.Job) error
	Get(ctx context.Context, id string) (types.Job, bool)
	Close()
}

package worker

import (
	"context"
	"errors"
	"math/rand/v2"
	"sync"
	"time"

	"go.uber.org/zap"

	"go-channel/internal/queue"
	"go-channel/internal/types"
)

// Handler processes a job; return an error to trigger retries.
type Handler func(ctx context.Context, job types.Job) error

type BackoffConfig struct {
	Base   time.Duration
	Max    time.Duration
	Jitter time.Duration
}

type Config struct {
	Concurrency int
	JobTimeout  time.Duration
	Backoff     BackoffConfig
}

type Pool struct {
	queue   queue.Queue
	handler Handler
	config  Config
	log     *zap.Logger

	wg      sync.WaitGroup
	cancel  context.CancelFunc
	started bool
	mu      sync.Mutex
}

func NewPool(q queue.Queue, h Handler, cfg Config, log *zap.Logger) *Pool {
	if cfg.Concurrency <= 0 {
		cfg.Concurrency = 1
	}
	if cfg.JobTimeout <= 0 {
		cfg.JobTimeout = 10 * time.Second
	}
	if cfg.Backoff.Base <= 0 {
		cfg.Backoff.Base = 500 * time.Millisecond
	}
	if cfg.Backoff.Max <= 0 {
		cfg.Backoff.Max = 5 * time.Second
	}
	return &Pool{
		queue:   q,
		handler: h,
		config:  cfg,
		log:     log.Named("worker_pool"),
	}
}

func (p *Pool) Start(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.started {
		return errors.New("pool already started")
	}
	ctx, cancel := context.WithCancel(ctx)
	p.cancel = cancel
	for i := 0; i < p.config.Concurrency; i++ {
		p.wg.Add(1)
		go p.runWorker(ctx, i)
	}
	p.started = true
	return nil
}

func (p *Pool) Stop() {
	p.mu.Lock()
	defer p.mu.Unlock()
	if !p.started {
		return
	}
	p.cancel()
	p.wg.Wait()
	p.started = false
}

func (p *Pool) runWorker(ctx context.Context, idx int) {
	defer p.wg.Done()
	logger := p.log.With(zap.Int("worker", idx))

	for {
		job, err := p.queue.Dequeue(ctx)
		if err != nil {
			if errors.Is(err, context.Canceled) || errors.Is(err, queue.ErrQueueClosed) {
				return
			}
			logger.Warn("dequeue failed", zap.Error(err))
			continue
		}

		jobCtx, cancel := context.WithTimeout(ctx, p.config.JobTimeout)
		err = p.handler(jobCtx, job)
		cancel()

		switch {
		case err == nil:
			_ = p.queue.Ack(ctx, job.ID)
			logger.Info("job succeeded", zap.String("job_id", job.ID), zap.Int("attempts", job.Attempts))
		case job.Attempts >= job.MaxAttempts:
			failedJob, _ := p.queue.Fail(ctx, job.ID, err)
			logger.Warn("job moved to dead letter", zap.String("job_id", failedJob.ID), zap.Error(err))
		default:
			backoff := p.backoffDuration(job.Attempts)
			time.Sleep(backoff)
			_ = p.queue.Requeue(ctx, job)
			logger.Info("job requeued", zap.String("job_id", job.ID), zap.Int("attempts", job.Attempts), zap.Duration("backoff", backoff))
		}
	}
}

func (p *Pool) backoffDuration(attempt int) time.Duration {
	d := p.config.Backoff.Base * time.Duration(attempt)
	if d > p.config.Backoff.Max {
		d = p.config.Backoff.Max
	}
	if p.config.Backoff.Jitter > 0 {
		j := time.Duration(rand.Int64N(int64(p.config.Backoff.Jitter)))
		d += j
	}
	return d
}


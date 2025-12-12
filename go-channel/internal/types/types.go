package types

import "time"

type JobStatus string

const (
	JobStatusQueued    JobStatus = "queued"
	JobStatusRunning   JobStatus = "running"
	JobStatusSucceeded JobStatus = "succeeded"
	JobStatusFailed    JobStatus = "failed"
	JobStatusDead      JobStatus = "dead_letter"
)

type EnqueueRequest struct {
	Payload     string `json:"payload"`
	MaxAttempts int    `json:"max_attempts,omitempty"`
}

type Job struct {
	ID          string    `json:"id"`
	Payload     string    `json:"payload"`
	Status      JobStatus `json:"status"`
	Attempts    int       `json:"attempts"`
	MaxAttempts int       `json:"max_attempts"`
	ErrMsg      string    `json:"error,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	NextAttempt time.Time `json:"next_attempt"`
}

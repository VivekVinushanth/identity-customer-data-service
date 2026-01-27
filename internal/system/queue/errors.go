package queue

import "errors"

var (
	ErrQueueFull       = errors.New("queue is full")
	ErrQueueNotStarted = errors.New("queue is not started")
	ErrUnknownProvider = errors.New("unknown queue provider")
	ErrTypeMismatch    = errors.New("queue provider registered with incompatible type")
)

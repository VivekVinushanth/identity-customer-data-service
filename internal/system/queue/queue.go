package queue

import "context"

type Handler[T any] func(ctx context.Context, msg T) error

type Queue[T any] interface {
	Start(ctx context.Context, handler Handler[T]) error
	Enqueue(ctx context.Context, msg T) error
	Close() error
	Name() string
}

type Factory[T any] func(cfg map[string]any) (Queue[T], error)

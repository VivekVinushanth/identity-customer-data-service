package memory

import (
	"context"
	"fmt"

	"github.com/wso2/identity-customer-data-service/internal/system/queue"
)

type MemoryQueue[T any] struct {
	name string
	ch   chan T
}

func New[T any](name string, size int) *MemoryQueue[T] {
	return &MemoryQueue[T]{name: name, ch: make(chan T, size)}
}

func (q *MemoryQueue[T]) Name() string { return q.name }

func (q *MemoryQueue[T]) Start(ctx context.Context, handler queue.Handler[T]) error {
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case msg, ok := <-q.ch:
				if !ok {
					return
				}
				_ = handler(ctx, msg) // handler logs errors; worker loop continues
			}
		}
	}()
	return nil
}

func (q *MemoryQueue[T]) Enqueue(ctx context.Context, msg T) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case q.ch <- msg:
		return nil
	default:
		return fmt.Errorf("%w: %s", queue.ErrQueueFull, q.name)
	}
}

func (q *MemoryQueue[T]) Close() error {
	close(q.ch)
	return nil
}

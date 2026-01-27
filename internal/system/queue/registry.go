package queue

import (
	"fmt"
	"sync"
)

var (
	mu        sync.RWMutex
	factories = map[string]any{} // provider -> typed factory stored as any
)

func Register[T any](provider string, factory Factory[T]) {
	mu.Lock()
	defer mu.Unlock()
	factories[provider] = factory
}

func New[T any](provider string, cfg map[string]any) (Queue[T], error) {
	mu.RLock()
	f, ok := factories[provider]
	mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrUnknownProvider, provider)
	}

	factory, ok := f.(Factory[T])
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrTypeMismatch, provider)
	}
	return factory(cfg)
}

package events

import (
	"sync"
)

type EventHandler[T any] func(T)

type Event[T any] struct {
	handlers []EventHandler[T]
	mu       sync.RWMutex
}

func NewEvent[T any]() *Event[T] {
	return &Event[T]{
		handlers: make([]EventHandler[T], 0),
		mu:       sync.RWMutex{},
	}
}

func Subscribe[T any](em *Event[T], handler EventHandler[T]) {
	em.mu.Lock()
	defer em.mu.Unlock()
	em.handlers = append(em.handlers, handler)
}

func Publish[T any](em *Event[T], args T) {
	em.mu.RLock()
	defer em.mu.RUnlock()
	for _, handler := range em.handlers {
		go handler(args)
	}
}

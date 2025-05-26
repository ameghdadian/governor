package governor

import (
	"context"
	"fmt"
	"sync/atomic"
)

type commandType int

const (
	read commandType = iota
	write
)

type command[T any] struct {
	kind     commandType
	response chan T
	writeFn  func(*T)
}

type ChanSync[T any] struct {
	commandCh chan command[T]
	closed    atomic.Bool
}

func New[T any](initial T) (*ChanSync[T], func()) {
	cs := ChanSync[T]{
		commandCh: make(chan command[T]),
	}

	go cs.loop(initial)

	cleanupFunc := func() {
		cs.closed.Store(true)
		close(cs.commandCh)
	}

	return &cs, cleanupFunc
}

func (cs *ChanSync[T]) loop(initial T) {
	state := initial
	for c := range cs.commandCh {
		switch c.kind {
		case read:
			c.response <- state
		case write:
			if c.writeFn != nil {
				c.writeFn(&state)
			}
		}
	}
}

func (cs *ChanSync[T]) Read() (T, error) {
	return cs.ReadWithContext(context.Background())
}

func (cs *ChanSync[T]) Write(f func(*T)) error {
	return cs.WriteWithContext(context.Background(), f)
}

func (cs *ChanSync[T]) ReadWithContext(ctx context.Context) (T, error) {
	var zero T

	if cs.closed.Load() {
		return zero, fmt.Errorf("unable to read after cleanup function called")
	}

	resp := make(chan T, 1)
	select {
	case cs.commandCh <- command[T]{kind: read, response: resp}:
	case <-ctx.Done():
		return zero, ctx.Err()
	}

	select {
	case r := <-resp:
		return r, nil
	case <-ctx.Done():
		return zero, ctx.Err()
	}
}

func (cs *ChanSync[T]) WriteWithContext(ctx context.Context, f func(*T)) error {
	if cs.closed.Load() {
		return fmt.Errorf("unable to write after cleanup function called")
	}

	cmd := command[T]{kind: write, writeFn: f}
	select {
	case cs.commandCh <- cmd:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

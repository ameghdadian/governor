// governor uses channels to synchronize access to shared state. It's built on top of Go's mantra:
//
// > "Don't communicate by sharing memory; instead, share memory by communicating."
// — Effective Go
//
// It uses a single goroutine to govern concurrent access to the shared resource—thus the name.
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

// ChanSync is the structure managing concurrent access to shared state.
type ChanSync[T any] struct {
	commandCh chan command[T]
	closed    atomic.Bool
}

// New returns a ChanSync object and returns a cleanup function to be called
// when we no longer need the object.
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

// Read returns the value of the shared state. It returns
// error if the cleanup function has already been called.
func (cs *ChanSync[T]) Read() (T, error) {
	return cs.ReadWithContext(context.Background())
}

// Write uses the callback function it's given to mutate
// the shared state.
func (cs *ChanSync[T]) Write(f func(*T)) error {
	return cs.WriteWithContext(context.Background(), f)
}

// ReadWithContext is the context-aware version of “Read“.
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

// WriteWithContext is the context-aware version of “Write“.
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

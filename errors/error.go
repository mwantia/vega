package errors

import (
	"errors"
	"sync"
)

type Errors struct {
	mu     sync.RWMutex
	errors []error
}

func (e *Errors) Add(err error) {
	if err == nil {
		return
	}

	e.mu.Lock()
	defer e.mu.Unlock()

	e.errors = append(e.errors, err)
}

func (e *Errors) Clear() {
	e.errors = make([]error, 0)
}

func (e *Errors) Empty() bool {
	e.mu.RLock()
	defer e.mu.RUnlock()

	return len(e.errors) == 0
}

func (e *Errors) Errors() error {
	e.mu.RLock()
	defer e.mu.RUnlock()

	if len(e.errors) == 0 {
		return nil
	}

	return errors.Join(e.errors...)
}

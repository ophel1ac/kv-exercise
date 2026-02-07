package kv

import (
	"errors"
	"fmt"
)

type ErrNotFound[T comparable] struct {
	Key T
}

func (e *ErrNotFound[T]) Error() string {
	return fmt.Sprintf("%v: value not found", e.Key)
}

// Чтобы работало errors.Is, errors.As
func (e *ErrNotFound[T]) Is(target error) bool {
	var errNotFound *ErrNotFound[T]
	ok := errors.As(target, &errNotFound)
	return ok
}

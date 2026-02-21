package kv

import (
	"sync"
)

// In memory Key Value data, нужен интерфейс к хранилищу чтобы в дальнейшем его можно было заменить на Redis или еще что нибудь

type KV[T comparable, V any] interface {
	Set(key T, value V)
	Get(key T) (V, error)
	Delete(key T)
}

type data[V any] struct {
	value V
	mu    sync.RWMutex
}

func newData[V any](value V) *data[V] {
	return &data[V]{
		value: value,
		mu:    sync.RWMutex{},
	}
}

type Storage[T comparable, V any] struct {
	data map[T]*data[V]
	mu   sync.RWMutex
}

func New[T comparable, V any]() *Storage[T, V] {
	return &Storage[T, V]{
		data: make(map[T]*data[V]),
		mu:   sync.RWMutex{},
	}
}

func (s *Storage[T, V]) Set(key T, value V) {
	s.mu.RLock()
	d, exists := s.data[key]
	s.mu.RUnlock()

	if exists {
		d.mu.Lock()
		d.value = value
		d.mu.Unlock()
	} else {
		s.mu.Lock()
		s.data[key] = newData(value)
		s.mu.Unlock()
	}
}

func (s *Storage[T, V]) Get(key T) (V, error) {
	s.mu.RLock()
	value, ok := s.data[key]
	s.mu.RUnlock()

	if !ok {
		var zero V
		return zero, &ErrNotFound[T]{Key: key}
	}

	// Иначе получаем race condition
	value.mu.RLock()
	v := value.value
	value.mu.RUnlock()

	return v, nil
}

func (s *Storage[T, V]) Delete(key T) {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.data, key)

}

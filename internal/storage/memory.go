package storage

import (
	"sync"
)

type memoryStorage struct {
	data map[string][]byte
	mu   sync.RWMutex
}

func newMemoryStorage() Storage {
	return &memoryStorage{data: make(map[string][]byte)}
}

func (s *memoryStorage) Write(id string, data []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	logf("[STORAGE] Write id %s", id)

	if _, ok := s.data[id]; ok {
		return ErrExists{}
	}

	s.data[id] = data

	return nil
}

func (s *memoryStorage) Get(id string) ([]byte, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	logf("[STORAGE] Get id %s", id)

	if data, ok := s.data[id]; !ok {
		return nil, ErrNotFound{}
	} else {
		return data, nil
	}
}

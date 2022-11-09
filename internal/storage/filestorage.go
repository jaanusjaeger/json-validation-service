package storage

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

type FileStorageConf struct {
	Dir string
}

type fileStorage struct {
	Dir string
	mu  sync.RWMutex
}

func newFileStorage(conf FileStorageConf) (Storage, error) {
	if conf.Dir == "" {
		return nil, fmt.Errorf("storage dir is not defined")
	}
	err := os.MkdirAll(conf.Dir, 0755)
	if err != nil {
		return nil, err
	}
	return &fileStorage{Dir: conf.Dir}, nil
}

func (s *fileStorage) Write(id string, data []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	logf("[STORAGE] Write id %s", id)

	path := filepath.Join(s.Dir, id)
	if _, err := os.Stat(path); err == nil {
		return ErrExists{}
	}

	return os.WriteFile(path, []byte(data), 0644)
}

func (s *fileStorage) Get(id string) ([]byte, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	logf("[STORAGE] Get id %s", id)

	path := filepath.Join(s.Dir, id)
	if data, err := os.ReadFile(path); os.IsNotExist(err) {
		return nil, ErrNotFound{}
	} else {
		return data, err
	}
}

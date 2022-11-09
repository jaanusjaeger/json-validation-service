package storage

import (
	"fmt"
	"log"
)

type Conf struct {
	File FileStorageConf
}

type ErrNotFound struct{}

func (e ErrNotFound) Error() string { return "not found" }

type ErrExists struct{}

func (e ErrExists) Error() string { return "already exists" }

type Storage interface {
	Write(id string, data []byte) error
	Get(id string) ([]byte, error)
}

func New(conf Conf) (Storage, error) {
	if conf.File != (FileStorageConf{}) {
		logf("Using file storage")
		return newFileStorage(conf.File)
	}
	logf("Using memory storage")
	return newMemoryStorage(), nil
}

func logf(format string, v ...any) {
	log.Printf("[STORAGE] %s", fmt.Sprintf(format, v...))
}

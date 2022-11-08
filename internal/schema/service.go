package schema

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/santhosh-tekuri/jsonschema/v5"
)

var NotFound = fmt.Errorf("not found")

type Conf struct {
	Dir string
}

type Service struct {
	Dir string
	mu  sync.RWMutex
}

func NewService(conf Conf) (*Service, error) {
	if conf.Dir == "" {
		return nil, fmt.Errorf("storage dir is not defined")
	}
	err := os.MkdirAll(conf.Dir, 0755)
	if err != nil {
		return nil, err
	}
	return &Service{Dir: conf.Dir}, nil
}

func (s *Service) CreateSchema(id string, schema []byte) error {
	_, err := jsonschema.CompileString("schema.json", string(schema))
	if err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	path := filepath.Join(s.Dir, id)
	if _, err = os.Stat(path); err == nil {
		return fmt.Errorf("file already exists: %s", path)
	}

	return os.WriteFile(path, []byte(schema), 0644)
}

func (s *Service) GetSchema(id string) ([]byte, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	path := filepath.Join(s.Dir, id)
	if data, err := os.ReadFile(path); os.IsNotExist(err) {
		return nil, NotFound
	} else {
		return data, err
	}
}

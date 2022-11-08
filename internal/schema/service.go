package schema

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/santhosh-tekuri/jsonschema/v5"
)

type ErrNotFound struct{}

func (e ErrNotFound) Error() string { return "schema not found" }

type ErrExists struct{}

func (e ErrExists) Error() string { return "schema already exists" }

type ErrInvalidFormat struct {
	err string
}

func (e ErrInvalidFormat) Error() string { return e.err }

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
	_, err := compileSchema(id+".json", schema)
	if err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	path := filepath.Join(s.Dir, id)
	if _, err = os.Stat(path); err == nil {
		return ErrExists{}
	}

	return os.WriteFile(path, []byte(schema), 0644)
}

func (s *Service) GetSchema(id string) ([]byte, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	path := filepath.Join(s.Dir, id)
	if data, err := os.ReadFile(path); os.IsNotExist(err) {
		return nil, ErrNotFound{}
	} else {
		return data, err
	}
}

func compileSchema(name string, bytes []byte) (*jsonschema.Schema, error) {
	sch, err := jsonschema.CompileString(name, string(bytes))
	if err != nil {
		var se *jsonschema.SchemaError
		if errors.As(err, &se) {
			return nil, ErrInvalidFormat{
				// Sanitize the error message - remove absolute path information
				err: strings.Replace(se.Error(), se.SchemaURL, name, -1),
			}
		}
		return nil, ErrInvalidFormat{
			err: err.Error(),
		}
	}
	return sch, nil
}

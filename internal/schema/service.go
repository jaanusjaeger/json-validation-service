package schema

import (
	"encoding/json"
	"errors"
	"strings"

	"github.com/jaanusjaeger/json-validation-service/internal/storage"
	"github.com/santhosh-tekuri/jsonschema/v5"
)

type ErrInvalidFormat struct {
	err string
}

func (e ErrInvalidFormat) Error() string { return e.err }

type Service struct {
	storage storage.Storage
}

func NewService(storage storage.Storage) *Service {
	return &Service{storage}
}

func (s *Service) CreateSchema(id string, schema []byte) error {
	_, err := compileSchema(id+".json", schema)
	if err != nil {
		return err
	}

	return s.storage.Write(id, schema)
}

func (s *Service) GetSchema(id string) ([]byte, error) {
	return s.storage.Get(id)
}

func (s *Service) ValidateJSON(jsonBytes []byte, id string) error {
	schBytes, err := s.GetSchema(id)
	if err != nil {
		return err
	}
	name := id + ".json"
	sch, err := compileSchema(name, schBytes)
	if err != nil {
		return err
	}

	var v interface{}
	if err := json.Unmarshal(jsonBytes, &v); err != nil {
		return ErrInvalidFormat{err: err.Error()}
	}

	cleanNulls(v)

	if err = sch.Validate(v); err != nil {
		var ve *jsonschema.ValidationError
		if errors.As(err, &ve) {
			return ErrInvalidFormat{
				// Sanitize the error message - remove absolute path information
				err: strings.Replace(ve.Error(), ve.AbsoluteKeywordLocation, name, -1),
			}
		}
		return err
	}

	return nil
}

func cleanNulls(i interface{}) {
	// According to https://pkg.go.dev/encoding/json@go1.19.2#Unmarshal:
	// * JSON object is stored in map[string]interface{}
	// * JSON array is stored in []interface{}
	switch v := i.(type) {
	case map[string]interface{}:
		nils := []string{}
		for key, value := range v {
			if value == nil {
				nils = append(nils, key)
			} else {
				cleanNulls(value)
			}
		}
		for _, key := range nils {
			delete(v, key)
		}
	case []interface{}:
		for _, elem := range v {
			// Not removing nils from JSON arrays, because that would change the
			// "structure" (number of elements) of the array.
			// In case of JSON object, the expected structure (class definition)
			// is usually known and removing a field value does not change it.
			cleanNulls(elem)
		}
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

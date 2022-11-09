package schema

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/jaanusjaeger/json-validation-service/internal/storage"
)

func TestPostSchema_ValidSchemaID_Success(t *testing.T) {
	testCases := []struct {
		url      string
		schemaID string
	}{
		{"/schema/1", "1"},
		{"/schema/schema1", "schema1"},
		{"/schema/schema_WITH-123", "schema_WITH-123"},
		{"/schema/schema1?query=allowed", "schema1"},
	}

	for _, tc := range testCases {
		t.Run(tc.url, func(t *testing.T) {
			h := mkHandler()
			req := httptest.NewRequest(http.MethodPost, tc.url, strings.NewReader("{}"))
			w := httptest.NewRecorder()

			h.schema(w, req)

			resp := w.Result()
			defer resp.Body.Close()
			expectResponse(t, http.StatusCreated, ActionUploadSchema, tc.schemaID, resp)
		})
	}
}

func TestPostSchema_InvalidSchemaID_Error(t *testing.T) {
	testCases := []struct {
		url      string
		schemaID string
	}{
		{"/schema", ""},
		{"/schema/", ""},
		{"/schema/.", "."},
		{"/schema/schema1/andmore", "schema1/andmore"},
	}

	for _, tc := range testCases {
		t.Run(tc.url, func(t *testing.T) {
			h := mkHandler()
			req := httptest.NewRequest(http.MethodPost, tc.url, strings.NewReader("{}"))
			w := httptest.NewRecorder()

			h.schema(w, req)

			resp := w.Result()
			defer resp.Body.Close()
			expectResponse(t, http.StatusBadRequest, ActionUploadSchema, tc.schemaID, resp)
		})
	}
}

func TestPostSchema_InvalidJSON_Error(t *testing.T) {
	h := mkHandler()
	req := httptest.NewRequest(http.MethodPost, "/schema/schema1", strings.NewReader("}{"))
	w := httptest.NewRecorder()

	h.schema(w, req)

	resp := w.Result()
	defer resp.Body.Close()
	expectResponse(t, http.StatusBadRequest, ActionUploadSchema, "schema1", resp)
}

func TestPostSchema_InvalidSchema_Error(t *testing.T) {
	h := mkHandler()
	req := httptest.NewRequest(http.MethodPost, "/schema/schema1", strings.NewReader(`{"type": "object-NOT"}`))
	w := httptest.NewRecorder()

	h.schema(w, req)

	resp := w.Result()
	defer resp.Body.Close()
	expectResponse(t, http.StatusBadRequest, ActionUploadSchema, "schema1", resp)
}

func TestPostSchema_MultipleTimesSameID_Error(t *testing.T) {
	h := mkHandler()
	prepareTestSchema(t, h)
	req := httptest.NewRequest(http.MethodPost, "/schema/schema1", strings.NewReader(schema1))
	w := httptest.NewRecorder()

	h.schema(w, req)

	resp := w.Result()
	defer resp.Body.Close()
	expectResponse(t, http.StatusConflict, ActionUploadSchema, "schema1", resp)
}

func TestGetSchema_Success(t *testing.T) {
	h := mkHandler()
	prepareTestSchema(t, h)
	req := httptest.NewRequest(http.MethodGet, "/schema/schema1", nil)
	w := httptest.NewRecorder()

	h.schema(w, req)

	resp := w.Result()
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("unexpected status: expected %d, got %d", http.StatusOK, resp.StatusCode)
	}
	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Errorf("unexpected error: %s", err)
	}
	var result any
	if err = json.Unmarshal(data, &result); err != nil {
		t.Errorf("unexpected error: %s", err)
	}
	var expect any
	if err = json.Unmarshal([]byte(schema1), &expect); err != nil {
		t.Errorf("unexpected error: %s", err)
	}
	if !reflect.DeepEqual(result, expect) {
		t.Errorf("unexpected schema: expected %d, got %d", expect, result)
	}
}

func TestGetSchema_UnknownSchemaID_Error(t *testing.T) {
	h := mkHandler()
	prepareTestSchema(t, h)
	req := httptest.NewRequest(http.MethodGet, "/schema/schema2", nil)
	w := httptest.NewRecorder()

	h.schema(w, req)

	resp := w.Result()
	defer resp.Body.Close()
	expectResponse(t, http.StatusNotFound, ActionDownloadSchema, "schema2", resp)
}

func mkHandler() *handler {
	storage, _ := storage.New(storage.Conf{})
	service := NewService(storage)
	return &handler{service}
}

func prepareTestSchema(t *testing.T, h *handler) {
	req := httptest.NewRequest(http.MethodPost, "/schema/schema1", strings.NewReader(schema1))
	w := httptest.NewRecorder()

	h.schema(w, req)

	resp := w.Result()
	defer resp.Body.Close()
	expectResponse(t, http.StatusCreated, ActionUploadSchema, "schema1", resp)
}

func expectResponse(t *testing.T, status int, action ActionType, id string, resp *http.Response) {
	t.Helper()

	if resp.StatusCode != status {
		t.Errorf("unexpected status: expected %d, got %d", status, resp.StatusCode)
	}
	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Errorf("unexpected error: %s", err)
	}
	var result Response
	if err = json.Unmarshal(data, &result); err != nil {
		t.Errorf("unexpected error: %s", err)
	}
	if result.Action != action {
		t.Errorf("unexpected action, expected: %s, got: %s", action, result.Action)
	}
	if result.ID != id {
		t.Errorf("unexpected id, expected: %s, got: %s", id, result.ID)
	}
	if status >= 200 && status < 300 && result.Status != StatusSuccess {
		t.Errorf("unexpected status, expected: %s, got: %s", StatusSuccess, result.Status)
	}
	if !(status >= 200 && status < 300) && result.Status != StatusError {
		t.Errorf("unexpected status, expected: %s, got: %s", StatusError, result.Status)
	}
}

const schema1 = `
{
	"$schema": "http://json-schema.org/draft-04/schema#",
	"type": "object",
	"properties": {
	  "source": {
		"type": "string"
	  },
	  "destination": {
		"type": "string"
	  },
	  "timeout": {
		"type": "integer",
		"minimum": 0,
		"maximum": 32767
	  },
	  "chunks": {
		"type": "array",
		"items": {
		  "type": "object",
		  "properties": {
			"size": {
			  "type": "integer"
			},
			"number": {
			  "type": "integer"
			}
		  },
		  "required": ["size"]
		}
	  }
	},
	"required": ["source", "destination"]
}`

const validJson1 = `
{
	"source": "/home/alice/image.iso",
	"destination": "/mnt/storage",
	"timeout": null,
	"chunks": [
		{
			"size": 1024,
			"number": null
		}
	]
}`

const invalidJson1 = `
{
	"source": "/home/alice/image.iso",
	"destination": "/mnt/storage",
	"timeout": null,
	"chunks": [
		null,
		{
			"size": 1024,
			"number": null
		}
	]
}`

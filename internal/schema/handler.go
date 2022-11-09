package schema

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"regexp"
	"strings"

	"github.com/jaanusjaeger/json-validation-service/internal/storage"
)

var schemaIDMatcher, _ = regexp.Compile("^[\\w-]+$")

type ActionType string

const (
	ActionUploadSchema   ActionType = "uploadSchema"
	ActionDownloadSchema            = "downloadSchema"
	ActionValidate                  = "validateDocument"
)

type StatusType string

const (
	StatusSuccess StatusType = "success"
	StatusError              = "error"
)

type Response struct {
	Action  ActionType `json:"action,omitempty"`
	ID      string     `json:"id,omitempty"`
	Status  StatusType `json:"status"`
	Message string     `json:"message,omitempty"`
}

func Handlers(service *Service) map[string]http.HandlerFunc {
	h := handler{service}

	return map[string]http.HandlerFunc{
		"/schema/":   h.schema,
		"/validate/": h.validate,
	}
}

type handler struct {
	service *Service
}

func (h *handler) schema(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		h.postSchema(w, r)
		return
	}
	if r.Method == http.MethodGet {
		h.getSchema(w, r)
		return
	}
	notFound(w, r)
}

func (h *handler) postSchema(w http.ResponseWriter, r *http.Request) {
	schemaID, err := getSchemaID(r.URL.Path)
	if err != nil {
		writeError(w, http.StatusBadRequest, ActionUploadSchema, schemaID, err)
		return
	}

	body, err := io.ReadAll(r.Body)
	defer r.Body.Close()
	if err != nil {
		writeError(w, http.StatusInternalServerError, ActionUploadSchema, schemaID, err)
		return
	}
	err = h.service.CreateSchema(schemaID, body)
	if err != nil {
		writeError(w, errStatus(err), ActionUploadSchema, schemaID, err)
		return
	}
	writeJSON(w, http.StatusCreated, Response{
		Action: ActionUploadSchema,
		ID:     schemaID,
		Status: StatusSuccess,
	})
}

func (h *handler) getSchema(w http.ResponseWriter, r *http.Request) {
	schemaID, err := getSchemaID(r.URL.Path)
	if err != nil {
		writeError(w, http.StatusBadRequest, ActionDownloadSchema, schemaID, err)
		return
	}

	sch, err := h.service.GetSchema(schemaID)
	if err != nil {
		writeError(w, errStatus(err), ActionDownloadSchema, schemaID, err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if _, err = w.Write(sch); err != nil {
		log.Println("error while writing JSON response")
	}
}

func (h *handler) validate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		notFound(w, r)
		return
	}

	schemaID, err := getSchemaID(r.URL.Path)
	if err != nil {
		writeError(w, http.StatusBadRequest, ActionValidate, schemaID, err)
		return
	}

	body, err := io.ReadAll(r.Body)
	defer r.Body.Close()
	if err != nil {
		writeError(w, http.StatusInternalServerError, ActionValidate, schemaID, err)
		return
	}

	if err = h.service.ValidateJSON(body, schemaID); err != nil {
		writeError(w, errStatus(err), ActionValidate, schemaID, err)
		return
	}
	writeJSON(w, http.StatusOK, Response{
		Action: ActionValidate,
		ID:     schemaID,
		Status: StatusSuccess,
	})
}

func getSchemaID(urlPath string) (string, error) {
	parts := strings.SplitN(urlPath, "/", 3)
	if len(parts) < 3 {
		return "", fmt.Errorf("schema ID is missing from the URL: %s", urlPath)
	}
	schemaID := parts[2]
	if !schemaIDMatcher.MatchString(schemaID) {
		return schemaID, fmt.Errorf("invalid schema ID format: %s", schemaID)
	}
	return schemaID, nil
}

func errStatus(err error) int {
	switch {
	case errors.As(err, &storage.ErrNotFound{}):
		return http.StatusNotFound
	case errors.As(err, &storage.ErrExists{}):
		return http.StatusConflict
	case errors.As(err, &ErrInvalidFormat{}):
		return http.StatusBadRequest
	}
	return http.StatusInternalServerError
}

func notFound(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusNotFound, Response{
		Status:  StatusError,
		Message: fmt.Sprintf("Not found: %s %s", r.Method, r.URL.Path),
	})
}

func writeError(w http.ResponseWriter, status int, action ActionType, schemaID string, err error) {
	writeJSON(w, status, Response{
		Action:  action,
		ID:      schemaID,
		Status:  StatusError,
		Message: err.Error(),
	})
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Println("error while writing JSON response")
	}
}

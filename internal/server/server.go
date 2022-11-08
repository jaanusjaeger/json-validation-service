package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/jaanusjaeger/json-validation-service/internal/schema"
)

var schemaIDMatcher, _ = regexp.Compile("^[\\w-]+$")

type Conf struct {
	Addr string
}

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

type Error struct {
	Error string
}

type Server struct {
	server http.Server
}

func New(conf Conf, service *schema.Service) *Server {
	mux := http.NewServeMux()
	h := handler{service: service}

	mux.HandleFunc("/schema/", h.schema)
	mux.HandleFunc("/", h.notFound)

	return &Server{
		server: http.Server{
			Addr:    conf.Addr,
			Handler: mux,
		},
	}
}

func (s *Server) ListenAndServe() error {
	return s.server.ListenAndServe()
}

func (s *Server) Shutdown(timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	return s.server.Shutdown(ctx)
}

type handler struct {
	service *schema.Service
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
	h.notFound(w, r)
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

func (h *handler) notFound(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusNotFound, Response{
		Status:  StatusError,
		Message: fmt.Sprintf("Not found: %s %s", r.Method, r.URL.Path),
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
	case errors.As(err, &schema.ErrNotFound{}):
		return http.StatusNotFound
	case errors.As(err, &schema.ErrExists{}):
		return http.StatusConflict
	case errors.As(err, &schema.ErrInvalidFormat{}):
		return http.StatusBadRequest
	}
	return http.StatusInternalServerError
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

package server

import (
	"context"
	"encoding/json"
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
	Action  ActionType `json:"action"`
	ID      string     `json:"id"`
	Status  StatusType `json:"status"`
	Message any        `json:"message,omitempty"`
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
	// TODO validation endpoint
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
	if r.Method != http.MethodPost && r.Method != http.MethodGet {
		h.notFound(w, r)
		return
	}

	schemaID, err := getSchemaID(r.URL.Path, "/schema/")
	if err != nil {
		writeJSON(w, http.StatusBadRequest, Response{
			Action:  ActionUploadSchema,
			ID:      schemaID,
			Status:  StatusError,
			Message: err.Error(),
		})
		return
	}

	if r.Method == http.MethodPost {
		body, err := io.ReadAll(r.Body)
		defer r.Body.Close()
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, Response{
				Action:  ActionUploadSchema,
				ID:      schemaID,
				Status:  StatusError,
				Message: err.Error(),
			})
			return
		}
		err = h.service.CreateSchema(schemaID, body)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, Response{
				Action:  ActionUploadSchema,
				ID:      schemaID,
				Status:  StatusError,
				Message: err.Error(),
			})
			return
		}
		writeJSON(w, http.StatusOK, Response{
			Action: ActionUploadSchema,
			ID:     schemaID,
			Status: StatusSuccess,
		})
		return
	}

	sch, err := h.service.GetSchema(schemaID)
	if err == schema.NotFound {
		writeJSON(w, http.StatusNotFound, Response{
			Action:  ActionDownloadSchema,
			ID:      schemaID,
			Status:  StatusError,
			Message: fmt.Errorf("schema not found: %s", schemaID),
		})
		return
	}
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, Response{
			Action:  ActionDownloadSchema,
			ID:      schemaID,
			Status:  StatusError,
			Message: err.Error(),
		})
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
		Action:  "",
		ID:      "",
		Status:  StatusError,
		Message: fmt.Sprintf("Not found: %s %s", r.Method, r.URL.Path),
	})
}

func getSchemaID(urlPath, prefix string) (string, error) {
	if !strings.HasPrefix(urlPath, prefix) || urlPath == prefix {
		return "", fmt.Errorf("schema ID is missing from the URL: %s", urlPath)
	}
	schemaID := urlPath[len(prefix):]
	if !schemaIDMatcher.MatchString(schemaID) {
		return schemaID, fmt.Errorf("invalid schema ID format: %s", schemaID)
	}
	return schemaID, nil
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Println("error while writing JSON response")
	}
}

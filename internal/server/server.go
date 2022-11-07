package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"
)

var schemaIDMatcher, _ = regexp.Compile("^[\\w-]+$")

type Conf struct {
	Addr string
}

type ActionType string

const (
	ActionSchema   ActionType = "uploadSchema"
	ActionValidate            = "validateDocument"
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

func New(conf Conf) *Server {
	mux := http.NewServeMux()
	h := handler{}

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
}

func (h *handler) schema(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost && r.Method != http.MethodGet {
		h.notFound(w, r)
		return
	}

	schemaID, err := getSchemaID(r.URL.Path, "/schema/")
	if err != nil {
		writeJSON(w, http.StatusBadRequest, Response{
			Action:  ActionSchema,
			ID:      schemaID,
			Status:  StatusError,
			Message: err.Error(),
		})
		return
	}

	// TODO
	writeJSON(w, http.StatusOK, Response{
		Action: ActionSchema,
		ID:     schemaID,
		Status: StatusSuccess,
	})
}

func (h *handler) notFound(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusNotFound, Response{
		Action:  ActionSchema,
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
		fmt.Println("error while writing JSON response")
	}
}
